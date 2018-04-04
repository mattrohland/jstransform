package test

// This file was autogenerated by https://github.com/GannettDigital/jstransform

type Complex struct {
	Caption string `json:"caption"`
	Credit  string `json:"credit"`
	Crops   []struct {
		Height       float64 `json:"height"`
		Name         string  `json:"name"`
		Path         string  `json:"path"`
		RelativePath string  `json:"relativePath"`
		Width        float64 `json:"width"`
	} `json:"crops"`
	Cutline        string `json:"cutline,omitempty"`
	DatePhotoTaken string `json:"datePhotoTaken"`
	Orientation    string `json:"orientation"`
	OriginalSize   struct {
		Height float64 `json:"height"`
		Width  float64 `json:"width"`
	} `json:"originalSize"`
	Type string `json:"type"`
	URL  struct {
		Absolute string `json:"absolute"`
		Publish  string `json:"publish"`
	} `json:"URL"`
}