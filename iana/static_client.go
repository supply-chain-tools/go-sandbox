package iana

import (
	"log"
	"os"
	"strings"
)

type staticClient struct {
}

func NewStaticClient() Client {
	return &staticClient{}
}

func (sc *staticClient) GetAllTlds() ([]string, error) {
	return tlds, nil
}

func generateStatic() error {
	remoteClient := NewRemoteClient()
	tlds, err := remoteClient.GetAllTlds()
	if err != nil {
		return err
	}

	sb := strings.Builder{}
	sb.WriteString("package iana\n\n")
	sb.WriteString("var tlds = []string{\n")

	for _, tld := range tlds {
		sb.WriteString("\t\"" + tld + "\",\n")
	}
	sb.WriteString("}\n")

	data := []byte(sb.String())

	err = os.WriteFile("static.go", data, 0644)
	if err != nil {
		log.Fatal(err)
	}

	return nil
}
