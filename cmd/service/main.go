package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
)

type Resolution struct {
	Width  int
	Height int
}

type ApiResponse struct {
	Products []struct {
		Images struct {
			Originals []string `json:"originals"`
		} `json:"images"`
	} `json:"products"`
}

func ResizeToWebP(img image.Image, res Resolution, quality int) ([]byte, error) {
	origBounds := img.Bounds()
	origWidth := origBounds.Dx()
	origHeight := origBounds.Dy()

	ratio := float64(res.Height) / float64(origHeight)
	newWidth := int(float64(origWidth) * ratio)

	resized := imaging.Resize(img, newWidth, res.Height, imaging.Lanczos)

	buf := new(bytes.Buffer)
	err := webp.Encode(buf, resized, &webp.Options{Quality: float32(quality)})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func main() {
	url := "https://realty.common.geo.paywb.com/api/realty/v1/catalog?id=1633996"
	resp, err := http.Get(url)
	if err != nil {
		panic(fmt.Errorf("failed to get API data: %w", err))
	}
	defer resp.Body.Close()

	var apiResp ApiResponse
	err = json.NewDecoder(resp.Body).Decode(&apiResp)
	if err != nil {
		return
	}

	if len(apiResp.Products) == 0 {
		fmt.Println("No products found")
		return
	}

	resolutions := []Resolution{
		{Width: 390, Height: 292},
		{Width: 390, Height: 644},
		{Width: 1428, Height: 775},
	}

	quality := 80

	for pIndex, product := range apiResp.Products {
		fmt.Printf("\nProduct #%d:\n", pIndex+1)

		if len(product.Images.Originals) == 0 {
			fmt.Println("  No images found for this product")
			continue
		}

		for i, imgURL := range product.Images.Originals {
			fmt.Printf("\nImage %d: %s\n", i+1, imgURL)

			imgResp, err := http.Get(imgURL)
			if err != nil {
				fmt.Printf("Error fetching image: %v\n", err)
				continue
			}
			imgBytes, err := io.ReadAll(imgResp.Body)
			imgResp.Body.Close()
			if err != nil {
				fmt.Printf("Error reading image data: %v\n", err)
				continue
			}

			img, _, err := image.Decode(bytes.NewReader(imgBytes))
			if err != nil {
				fmt.Printf("Error decoding image: %v\n", err)
				continue
			}

			bounds := img.Bounds()
			origWidth := bounds.Dx()
			origHeight := bounds.Dy()
			origSizeKB := float64(len(imgBytes)) / 1024.0

			fmt.Printf("Original: %.2f KB, resolution: %dx%d\n", origSizeKB, origWidth, origHeight)

			for _, res := range resolutions {
				converted, err := ResizeToWebP(img, res, quality)
				if err != nil {
					fmt.Printf("    Conversion error for %dx%d: %v\n", res.Width, res.Height, err)
					continue
				}
				sizeKB := float64(len(converted)) / 1024.0
				fmt.Printf("webp %dx%d, size: %.2f KB\n", res.Width, res.Height, sizeKB)
			}
		}
	}
}
