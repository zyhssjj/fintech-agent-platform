package communication

import (
	"archive/zip"
	"bytes"
)

func buildStaticDOCX(contentTypes, rels, docRels, documentXML string) []byte {
	buffer := &bytes.Buffer{}
	zw := zip.NewWriter(buffer)
	files := map[string]string{
		"[Content_Types].xml": contentTypes,
		"_rels/.rels":         rels,
		"word/_rels/document.xml.rels": docRels,
		"word/document.xml":   documentXML,
	}
	for name, content := range files {
		writer, err := zw.Create(name)
		if err != nil {
			return nil
		}
		if _, err := writer.Write([]byte(content)); err != nil {
			return nil
		}
	}
	if err := zw.Close(); err != nil {
		return nil
	}
	return buffer.Bytes()
}
