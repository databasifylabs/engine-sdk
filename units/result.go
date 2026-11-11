package units

func Failed(result string) bool {
	return result != "" && result != "success"
}
