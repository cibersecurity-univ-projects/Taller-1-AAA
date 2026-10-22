package service

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
)

const (
	// MaxImages es el máximo de imágenes por publicación.
	MaxImages = 12
	// MaxImageBytes es el tamaño máximo de cada imagen: 2 MB.
	MaxImageBytes = 2 << 20

	// maxImageSide limita el ancho y el alto para no decodificar imágenes
	// enormes que ocuparían demasiada memoria.
	maxImageSide = 4096
	jpegQuality  = 85
)

// cleanImage comprueba que los bytes sean realmente un JPEG o un PNG (por su
// contenido, no por su extensión), de tamaño razonable, y los codifica de
// nuevo. Al volver a codificar se descartan los metadatos, incluida la
// ubicación GPS. Devuelve la imagen limpia y su extensión.
func cleanImage(raw []byte) ([]byte, string, error) {
	if len(raw) == 0 || len(raw) > MaxImageBytes {
		return nil, "", ErrInvalidImage
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || (format != "jpeg" && format != "png") {
		return nil, "", ErrInvalidImage
	}
	if config.Width < 1 || config.Height < 1 || config.Width > maxImageSide || config.Height > maxImageSide {
		return nil, "", ErrInvalidImage
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, "", ErrInvalidImage
	}

	var out bytes.Buffer
	if format == "jpeg" {
		if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: jpegQuality}); err != nil {
			return nil, "", err
		}
		return out.Bytes(), ".jpg", nil
	}
	if err := png.Encode(&out, img); err != nil {
		return nil, "", err
	}
	return out.Bytes(), ".png", nil
}

// Las imágenes de cada publicación viven en una carpeta propia: imágenes/<id>/.
// Los nombres los genera el servidor (1.jpg, 2.png...) y solo se leen los que la
// publicación tiene registrados, así que nunca llegan rutas del cliente.

func (s *Vehicles) imageFolder(vehicleID int) string {
	return filepath.Join(s.imagesDir, strconv.Itoa(vehicleID))
}

func (s *Vehicles) saveImage(vehicleID int, name string, data []byte) error {
	folder := s.imageFolder(vehicleID)
	if err := os.MkdirAll(folder, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(folder, name), data, 0o600)
}

func (s *Vehicles) readImage(vehicleID int, name string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Join(s.imageFolder(vehicleID), name))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	return data, err
}

// removeImages borra la carpeta de la publicación; si no existe, no es un error.
func (s *Vehicles) removeImages(vehicleID int) error {
	return os.RemoveAll(s.imageFolder(vehicleID))
}
