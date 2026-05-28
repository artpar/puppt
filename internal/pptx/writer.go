package pptx

import (
	"archive/zip"
	"context"
	"os"
)

// Write writes the package parts to a .pptx ZIP archive. It preserves part
// bytes supplied in Package.Parts and only changes ZIP container metadata.
func Write(ctx context.Context, pkg *Package, outputPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return packageError(ErrorInvalidPackage, "create", outputPath, "", err)
	}
	defer file.Close()

	archive := zip.NewWriter(file)
	for _, partName := range pkg.PartNames() {
		if err := ctx.Err(); err != nil {
			archive.Close()
			return err
		}
		writer, err := archive.Create(partName)
		if err != nil {
			archive.Close()
			return packageError(ErrorInvalidPackage, "write", outputPath, partName, err)
		}
		if _, err := writer.Write(pkg.Parts[partName]); err != nil {
			archive.Close()
			return packageError(ErrorInvalidPackage, "write", outputPath, partName, err)
		}
	}
	if err := archive.Close(); err != nil {
		return packageError(ErrorInvalidPackage, "close", outputPath, "", err)
	}
	return nil
}
