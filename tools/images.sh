#!/usr/bin/env bash
set -euo pipefail

tools=$(cd "$(dirname "$0")" && pwd)
root=$PWD
versions=versions.json
sdk_module=github.com/databasifylabs/engine-sdk
content_label=io.databasify.content-hash
reusable_builds=30
docker_hub=registry-1.docker.io

usage() {
    cat >&2 <<'EOF'
usage: images.sh [--versions <file>] <command>

Run from the engine repository root; --versions defaults to versions.json there.

  keys <engine>                 the engine's image keys as a JSON array (CI matrix)
  args <key>                    build arguments as KEY=VALUE lines
  hash <key>                    content hash of everything the image is built from
  plan <key> <commit> <default> action=, hash=, engine=, repository=, version=, source=, tags= for CI;
                                default=true (the default branch) also moves the <version>, <line> and latest tags
  tag <repository> <tags> <source...>
                                point the comma separated tags at the sources (DRY_RUN=true only prints)
  build <key> [buildx args...]  build the image locally from the repository
  smoke <image> <engine>        tool contract, info and engine start/stop checks
  refresh [key...]              move base images, copied images, packages and Go modules to their newest
                                patch within the pinned line, then print the changes as Markdown
  newest <key>                  the newest base tag read from stdin that stays inside the key's line
  check                         fails when a key is not <engine>-<line>, its base tag left the line
                                or the sdk pin is not the SDK version go.mod requires
EOF
    exit 64
}

field() {
    jq -er --arg key "$1" ".images[\$key]$2" "$versions"
}

engine_of() {
    field "$1" .engine
}

sha() {
    sha256sum | cut -c1-64
}

go_module() {
    (cd "$root" && go mod edit -json)
}

sdk_required() {
    go_module | jq -er --arg module "$sdk_module" '.Require[]? | select(.Path == $module) | .Version | ltrimstr("v")'
}

local_replacements() {
    go_module | jq -r '.Replace[]? | select(.New.Version == null) | "\(.Old.Path) \(.New.Path)"'
}

replaced_dirs() {
    local module path
    while read -r module path; do
        [[ $path =~ ^\.\./[^/]+$ ]] \
            || { echo "go.mod replaces $module with $path; images.sh only builds sibling checkouts written as ../<dir>" >&2; return 1; }
        (cd "$root" && cd "$path" 2> /dev/null && [ -f go.mod ] && pwd) \
            || { echo "go.mod replaces $module with $path, which is not a module checkout" >&2; return 1; }
    done < <(local_replacements)
}

go_files_in() {
    (cd "$1" && find . -path ./.git -prune -o -type f \( -name go.mod -o -name go.sum -o -name '*.go' \) ! -name '*_test.go' -print | LC_ALL=C sort)
}

go_sources_in() {
    (cd "$1" && go_files_in . | xargs cat) | sha
}

sources_hash() {
    local dirs dir
    dirs=$(replaced_dirs) || return 1
    {
        go_sources_in "$root"
        for dir in $dirs; do
            go_sources_in "$dir"
        done
    } | sha
}

stage_modules() {
    local staging=$1 dirs dir
    dirs=$(replaced_dirs) || return 1
    for dir in $dirs; do
        mkdir -p "$staging/${dir##*/}"
        go_files_in "$dir" | (cd "$dir" && tar -cf - -T -) | tar -xf - -C "$staging/${dir##*/}"
    done
}

cmd_keys() {
    jq -c --arg engine "$1" '[.images | to_entries[] | select(.value.engine == $engine) | .key] | sort' "$versions"
}

cmd_args() {
    field "$1" "" > /dev/null
    jq -r --arg key "$1" --arg sdk "$(jq -er .sdk "$versions")" '
        def ref: "\(.image):\(.tag)@\(.digest)";
        def argname: ascii_upcase | gsub("[^A-Z0-9]"; "_");
        .go as $go | .images[$key] as $image |
        "GO_IMAGE=\($go | ref)",
        "BASE_IMAGE=\($image.base | ref)",
        "RELEASE=\($image.release)",
        "LINE=\($image.line)",
        "SDK_VERSION=\($sdk)",
        "PACKAGES=\([$image.packages // {} | to_entries[] | "\(.key)=\(.value)"] | join(" "))",
        ($image.copies // {} | to_entries[] | "\(.key | argname)_IMAGE=\(.value | ref)"),
        ($image.modules // {} | to_entries[] | "\(.key | argname)_VERSION=\(.value.version)"),
        ($image.sources // {} | to_entries[] | (.key | argname) as $name |
            "\($name)_VERSION=\(.value.version)",
            "\($name)_SHA256=\(.value.sha256)")
    ' "$versions"
}

image_files_of() {
    local dir="$root/images/$1"
    printf '%s\n' "$dir/Dockerfile"
    [ ! -f "$dir/repositories.sh" ] || printf '%s\n' "$dir/repositories.sh"
}

cmd_hash() {
    local engine sources files pins
    engine=$(engine_of "$1") || return 1
    sources=$(sources_hash) || return 1
    files=$(image_files_of "$engine" | xargs cat | sha) || return 1
    pins=$(jq -cS --arg key "$1" '{registry, sdk, go, image: .images[$key]}' "$versions") || return 1
    printf '%s\n%s\n%s\n' "$sources" "$files" "$pins" | sha
}

registry_tags() {
    local repository=$1 host=${1%%/*} name=${1#*/} challenge realm service token url status next
    local body headers
    challenge=$(curl -sS -o /dev/null -D - "https://$host/v2/" | tr -d '\r' | sed -n 's/^[Ww][Ww][Ww]-[Aa]uthenticate: *//p')
    realm=$(sed -n 's/.*realm="\([^"]*\)".*/\1/p' <<< "$challenge")
    service=$(sed -n 's/.*service="\([^"]*\)".*/\1/p' <<< "$challenge")
    token=""
    if [ -n "$realm" ] && ! token=$(curl -fsS --get --data-urlencode "service=$service" --data-urlencode "scope=repository:$name:pull" "$realm" | jq -er '.token // .access_token'); then
        echo "$repository is not readable anonymously yet, treating it as unpublished" >&2
        return 0
    fi
    local auth=()
    [ -z "$token" ] || auth=(-H "Authorization: Bearer $token")
    body=$(mktemp)
    headers=$(mktemp)
    url="https://$host/v2/$name/tags/list?n=1000"
    while [ -n "$url" ]; do
        status=$(curl -sS -o "$body" -D "$headers" -w '%{http_code}' "${auth[@]}" "$url")
        [ "$status" != 404 ] || break
        [ "$status" = 200 ] || { echo "listing the tags of $repository answered $status" >&2; rm -f "$body" "$headers"; return 1; }
        jq -r '.tags // [] | .[]' "$body"
        next=$(tr -d '\r' < "$headers" | sed -n 's/^[Ll]ink: *<\([^>]*\)>.*rel="\{0,1\}next.*/\1/p')
        url=${next:+https://$host$next}
    done
    rm -f "$body" "$headers"
}

published_label() {
    docker buildx imagetools inspect "$1" --format '{{json .Image}}' 2> /dev/null \
        | jq -r --arg label "$2" 'if has("config") then . else (to_entries | first | .value) end | .config.Labels[$label] // ""' \
        || true
}

published_digest() {
    docker buildx imagetools inspect "$1" --format '{{json .Manifest}}' | jq -er .digest
}

repository_of() {
    printf '%s/databasify-%s' "$(jq -er .registry "$versions")" "$(engine_of "$1")"
}

newest_release_of() {
    jq -r --arg engine "$1" '.images[] | select(.engine == $engine) | .release' "$versions" | sort -V | tail -n 1
}

moving_tags_of() {
    local key=$1 release line
    release=$(field "$key" .release)
    line=$(field "$key" .line)
    printf '%s,%s' "$release" "$line"
    [ "$release" != "$(newest_release_of "$(engine_of "$key")")" ] || printf ',latest'
}

build_with_hash() {
    local repository=$1 release=$2 hash=$3 own_tag=$4 tag
    while IFS= read -r tag; do
        if [ "$(published_label "$repository:$tag" "$content_label")" = "$hash" ]; then
            published_digest "$repository:$tag"
            return 0
        fi
    done < <({
        echo "$own_tag"
        registry_tags "$repository" | grep -E "^${release//./\\.}-[0-9a-f]{7}\$" | grep -vx "$own_tag" | tail -n "$reusable_builds"
    } | awk '!seen[$0]++')
    return 1
}

cmd_plan() {
    local key=$1 commit=$2 default_branch=$3 release repository hash build_tag tags action=build source=""
    release=$(field "$key" .release)
    repository=$(repository_of "$key")
    hash=$(cmd_hash "$key")
    build_tag="$release-${commit:0:7}"
    tags=$build_tag
    [ "$default_branch" != true ] || tags="$tags,$(moving_tags_of "$key")"
    if [ "$(published_label "$repository:$release" "$content_label")" = "$hash" ]; then
        action=skip
    elif source=$(build_with_hash "$repository" "$release" "$hash" "$build_tag"); then
        action=retag
    fi
    printf 'action=%s\nhash=%s\nengine=%s\nrepository=%s\nversion=%s\nsource=%s\ntags=%s\n' \
        "$action" "$hash" "$(engine_of "$key")" "$repository" "$release" "$source" "$tags"
}

cmd_tag() {
    local repository=$1 tags=$2 tag
    shift 2
    local args=()
    [ "${DRY_RUN:-false}" != true ] || args+=(--dry-run)
    for tag in ${tags//,/ }; do
        args+=(--tag "$repository:$tag")
    done
    docker buildx imagetools create "${args[@]}" "$@"
}

cmd_build() {
    local key=$1 engine line hash
    shift
    engine=$(engine_of "$key")
    hash=$(cmd_hash "$key")
    modules=$(mktemp -d)
    trap 'rm -rf "$modules"' EXIT
    stage_modules "$modules"
    local args=()
    while IFS= read -r line; do
        args+=(--build-arg "$line")
    done < <(cmd_args "$key")
    docker buildx build "${args[@]}" --build-context "modules=$modules" --label "$content_label=$hash" --file "$root/images/$engine/Dockerfile" "$@" "$root"
}

cmd_smoke() {
    local engine=$2 checks="$root/images/$2/smoke" info code
    smoke_image=$1
    smoke_container="databasify-smoke-$engine-$$"
    echo "--- databasify info"
    info=$(docker run --rm --entrypoint databasify "$smoke_image" info)
    echo "$info"
    grep -qx "engine=$engine" <<< "$info"
    echo "--- tool contract"
    docker run --rm --entrypoint bash "$smoke_image" -c '
        for tool in bash ps ss ip jq openssl curl; do
            command -v "$tool" > /dev/null || { echo "missing $tool"; exit 1; }
        done
        echo "bash ps ss ip jq openssl curl present"'
    echo "--- engine start"
    smoke_work=$(mktemp -d)
    chmod 755 "$smoke_work"
    trap smoke_cleanup EXIT
    mkdir "$smoke_work/conf"
    cp "$checks/processes.json" "$smoke_work/conf/processes.json"
    [ ! -f "$checks/postgresql.conf" ] || cp "$checks/postgresql.conf" "$smoke_work/conf/postgresql.conf"
    docker run -d --name "$smoke_container" --volume "$smoke_work:/srv/$engine" "$smoke_image" > /dev/null
    if ! docker exec -i "$smoke_container" databasify step smoke < "$checks/check.sh"; then
        docker logs "$smoke_container" 2>&1 | tail -50
        exit 1
    fi
    if [ -e "$root/images/$engine/conformance" ]; then
        run_conformance
    fi
    echo "--- docker stop"
    docker stop --time 60 "$smoke_container" > /dev/null
    code=$(docker inspect --format '{{.State.ExitCode}}' "$smoke_container")
    docker logs "$smoke_container" 2>&1 | grep 'databasify:' | tail -8
    [ "$code" = 0 ] || { echo "container exited with $code"; exit 1; }
    echo "smoke passed for $smoke_image"
}

run_conformance() {
    local result
    echo "--- engine api conformance"
    result=$(cd "$root" && DATABASIFY_CONFORMANCE_CONTAINER=$smoke_container go test -count=1 -v -run TestConformance github.com/databasifylabs/engine-sdk/conformance 2>&1) || { echo "$result"; exit 1; }
    echo "$result"
    grep -q -e '--- PASS: TestConformance' <<< "$result" || { echo "conformance did not run to a pass"; exit 1; }
}

smoke_cleanup() {
    docker rm -f "$smoke_container" > /dev/null 2>&1 || true
    docker run --rm --entrypoint bash --volume "$smoke_work:/work" "$smoke_image" -c 'rm -rf /work/* /work/.[!.]*' > /dev/null 2>&1 || true
    rmdir "$smoke_work" 2> /dev/null || true
}

escape() {
    sed 's/[][\.*^$+?(){}|/]/\\&/g' <<< "$1"
}

dots_in() {
    tr -cd . <<< "$1" | wc -c | tr -d ' '
}

update() {
    local filter=$1 updated
    shift
    updated=$(mktemp)
    jq "$@" "$filter" "$versions" > "$updated"
    cat "$updated" > "$versions"
    rm -f "$updated"
}

registry_path_of() {
    local image=$1 first=${1%%/*}
    if [ "$first" = "$image" ]; then
        printf '%s/library/%s' "$docker_hub" "$image"
    elif [[ $first == *[.:]* ]]; then
        printf '%s' "$image"
    else
        printf '%s/%s' "$docker_hub" "$image"
    fi
}

newest_matching() {
    grep -E "$1" | sort -V | tail -n 1 || true
}

newest_tag_of() {
    registry_tags "$(registry_path_of "$1")" | newest_matching "$2"
}

patch_pattern_of() {
    [[ $1 =~ ^(.*\.)([0-9]+)(.*)$ ]] || { echo "$1 has no patch number" >&2; return 1; }
    printf '^%s[0-9]+%s$' "$(escape "${BASH_REMATCH[1]}")" "$(escape "${BASH_REMATCH[3]}")"
}

release_pattern_of() {
    local key=$1 line release suffix
    line=$(field "$key" .line)
    release=$(field "$key" .release)
    suffix=$(field "$key" .base.tag)
    suffix=${suffix#"$release"}
    printf '^%s(\\.[0-9]+){%d}%s$' "$(escape "$line")" "$(( $(dots_in "$release") - $(dots_in "$line") ))" "$(escape "$suffix")"
}

pin_image() {
    local path=$1 tag=$2 image digest
    image=$(jq -er "$path.image" "$versions")
    digest=$(published_digest "$image:$tag")
    update "$path.tag = \$tag | $path.digest = \$digest" --arg tag "$tag" --arg digest "$digest"
}

refresh_pinned_image() {
    local path=$1 image tag newest
    image=$(jq -er "$path.image" "$versions")
    tag=$(jq -er "$path.tag" "$versions")
    newest=$(newest_tag_of "$image" "$(patch_pattern_of "$tag")")
    pin_image "$path" "${newest:?no tag of $image matches $tag}"
}

cmd_newest() {
    newest_matching "$(release_pattern_of "$1")"
}

cmd_check() {
    local key expected tag sdk required failed=0
    sdk=$(jq -er .sdk "$versions")
    required=$(sdk_required) || { echo "go.mod does not require $sdk_module" >&2; return 1; }
    [ "$sdk" = "$required" ] || { echo "the sdk pin $sdk is not $required, the version go.mod requires" >&2; failed=1; }
    for key in $(jq -r '.images | keys[]' "$versions"); do
        expected="$(engine_of "$key")-$(field "$key" .line)"
        tag=$(field "$key" .base.tag)
        [ "$key" = "$expected" ] || { echo "$key: the key must be $expected" >&2; failed=1; }
        grep -Eq "$(release_pattern_of "$key")" <<< "$tag" || { echo "$key: base tag $tag left line $(field "$key" .line)" >&2; failed=1; }
    done
    return "$failed"
}

refresh_base() {
    local key=$1 path=".images[\"$1\"]" release tag newest
    release=$(field "$key" .release)
    tag=$(field "$key" .base.tag)
    newest=$(newest_tag_of "$(field "$key" .base.image)" "$(release_pattern_of "$key")")
    pin_image "$path.base" "${newest:?no release of $key is published}"
    update "$path.release = \$release" --arg release "${newest%"${tag#"$release"}"}"
}

refresh_copies() {
    local key=$1 name
    for name in $(jq -r --arg key "$key" '.images[$key].copies // {} | keys[]' "$versions"); do
        refresh_pinned_image ".images[\"$key\"].copies[\"$name\"]"
    done
}

refresh_packages() {
    local key=$1 base name version
    local names=()
    while IFS= read -r name; do
        names+=("$name")
    done < <(jq -r --arg key "$key" '.images[$key].packages // {} | keys[]' "$versions")
    [ ${#names[@]} -gt 0 ] || return 0
    base=$(jq -er --arg key "$key" '.images[$key].base | "\(.image):\(.tag)@\(.digest)"' "$versions")
    while IFS='=' read -r name version; do
        if [ -z "$version" ] || [ "$version" = "(none)" ]; then
            echo "$key: no installable version of $name" >&2
            return 1
        fi
        update ".images[\$key].packages[\$name] = \$version" --arg key "$key" --arg name "$name" --arg version "$version"
    done < <(docker run --rm --entrypoint bash --volume "$tools:/tools:ro" --volume "$root/images/$(engine_of "$key"):/engine:ro" "$base" /tools/packages.sh /engine "${names[@]}")
}

newest_module_version_of() {
    local module=$1 current=$2
    curl -fsS "https://proxy.golang.org/$module/@v/list" \
        | grep -E "^v${current%%.*}\.[0-9]+\.[0-9]+\$" | sort -V | tail -n 1 | sed 's/^v//'
}

refresh_modules() {
    local key=$1 name module current newest
    for name in $(jq -r --arg key "$key" '.images[$key].modules // {} | keys[]' "$versions"); do
        module=$(field "$key" ".modules[\"$name\"].module")
        current=$(field "$key" ".modules[\"$name\"].version")
        newest=$(newest_module_version_of "$module" "$current")
        update ".images[\$key].modules[\$name].version = \$version" --arg key "$key" --arg name "$name" --arg version "${newest:?no release of $module}"
    done
}

changes_since() {
    jq -rn --slurpfile before "$1" --slurpfile after "$versions" '
        def short: if type == "string" and startswith("sha256:") then .[0:19] else . end;
        $after[0] | paths(scalars) as $path
        | ($before[0] | getpath($path)) as $old
        | ($after[0] | getpath($path)) as $new
        | select($old != $new)
        | "- `\($path | map(tostring) | join("."))`: `\($old | short)` → `\($new | short)`"'
}

cmd_refresh() {
    local before key
    [ $# -gt 0 ] || while IFS= read -r key; do set -- "$@" "$key"; done < <(jq -r '.images | keys[]' "$versions")
    for key in "$@"; do
        field "$key" "" > /dev/null || { echo "unknown image key $key" >&2; return 64; }
    done
    before=$(mktemp)
    cp "$versions" "$before"
    refresh_pinned_image .go
    for key in "$@"; do
        refresh_base "$key"
        refresh_copies "$key"
        refresh_packages "$key"
        refresh_modules "$key"
    done
    changes_since "$before"
    rm -f "$before"
    cmd_check
}

if [ "${1:-}" = --versions ]; then
    [ $# -ge 2 ] || usage
    versions=$2
    shift 2
fi
[ $# -ge 1 ] || usage
command=$1
shift
case "$command" in
    keys) [ $# -eq 1 ] || usage; cmd_keys "$1" ;;
    args) [ $# -eq 1 ] || usage; cmd_args "$1" ;;
    hash) [ $# -eq 1 ] || usage; cmd_hash "$1" ;;
    plan) [ $# -eq 3 ] || usage; cmd_plan "$@" ;;
    tag) [ $# -ge 3 ] || usage; cmd_tag "$@" ;;
    build) [ $# -ge 1 ] || usage; cmd_build "$@" ;;
    smoke) [ $# -eq 2 ] || usage; cmd_smoke "$@" ;;
    refresh) cmd_refresh "$@" ;;
    newest) [ $# -eq 1 ] || usage; cmd_newest "$1" ;;
    check) [ $# -eq 0 ] || usage; cmd_check ;;
    *) usage ;;
esac
