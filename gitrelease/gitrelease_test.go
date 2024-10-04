package gitrelease

import "testing"

func TestSignaturePayload(t *testing.T) {
	pae, err := PreAuthenticationEncoding([]byte("testPayload"))
	if err != nil {
		t.Fatal(err)
	}

	if string(pae) != "DSSEv1 28 application/vnd.in-toto+json 11 testPayload" {
		t.Errorf("got %s, expected 'DSSEv1 28 application/vnd.in-toto+json 11 testPayload'", string(pae))
	}
}
