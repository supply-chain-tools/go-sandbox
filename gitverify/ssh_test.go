package gitverify

import (
	"testing"
)

func TestSSSH(t *testing.T) {
	data := "foo"
	namespace := "file"

	type TestCase struct {
		Name      string
		Key       string
		Signature string
	}

	testCases := []TestCase{
		{
			Name:      "ssh-ed25519",
			Key:       "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIIAQv90+kSOSKZYlMoWO0eX6QZ1Nt5n2BviA4vFx3lgK",
			Signature: "-----BEGIN SSH SIGNATURE-----\nU1NIU0lHAAAAAQAAADMAAAALc3NoLWVkMjU1MTkAAAAggBC/3T6RI5IpliUyhY7R5fpBnU\n23mfYG+IDi8XHeWAoAAAAEZmlsZQAAAAAAAAAGc2hhNTEyAAAAUwAAAAtzc2gtZWQyNTUx\nOQAAAEB8BS7mn2DJvRy0mbdXJN3nDSIN2pfBU/1UgM5kpGkCO1vRqyncLEd8/BEGgPBNhw\nqiV76Q1s5vE9wEidPdKMQI\n-----END SSH SIGNATURE-----\n",
		},
		{
			Name:      "ecdsa-sha2-nistp256",
			Key:       "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBH2r8kV3iq50ugWjL3l4OaLEhGNUhMPc/A2UWQSix/I5XEG6sfnXZre06ROUF2DaWxiACUiLhO1UDUY0guun3ZQ=",
			Signature: "-----BEGIN SSH SIGNATURE-----\nU1NIU0lHAAAAAQAAAGgAAAATZWNkc2Etc2hhMi1uaXN0cDI1NgAAAAhuaXN0cDI1NgAAAE\nEEfavyRXeKrnS6BaMveXg5osSEY1SEw9z8DZRZBKLH8jlcQbqx+ddmt7TpE5QXYNpbGIAJ\nSIuE7VQNRjSC66fdlAAAAARmaWxlAAAAAAAAAAZzaGE1MTIAAABkAAAAE2VjZHNhLXNoYT\nItbmlzdHAyNTYAAABJAAAAIQCU6xom5vuvd0I7dTSSC2smbCiob19xJvvIMgx7GHwnwgAA\nACAdRONfcC9dGxpthNvG0EfTYF+yXPFhRKDKiENny7kkkw==\n-----END SSH SIGNATURE-----\n",
		},
		{
			Name:      "ssh-rsa 4096",
			Key:       "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDgzDqsPb9NReMvYOHtrVj+sQcgRBubWkgB304L303uB4qpFRMSMRAmMIHV0W8LFql2tNo5UUoUy9vozL7Q0HqoLrA6suJ7OPH6slp6xBfoVPEYy8TcivDrM4Ri1rmloXCXDfydetuy4AOC7EA9ZxHJDAR9fGdgTMJtojTS2p/z3towkNUj3tDzFnatGI695klwCuB6jK+zsSkJ4uohWC11LdzKl2bTbqo2bQ6Ebu4kXwlaK/jcjpGaPqFG9E9P+tarc5zphzgzYZLAkCKCzBYKslYeVxPtsjq4n5QsNstBJcTWC+41I7e656sCqUyB759YveT1yBPoeCPUiuegmcNgKcwA3d+9PBUoGxOXUG9/WAODJUloae0Y1Lury123JYo5U1j5xtrxp6xNflt3/I3l7v8fszdE/5vTNXeKRlVQ1tBdAG77ejgGGojkM6l14Wma1+HkF4NZEaTtoAn8tCygxxr9bAB4tAc4pDJclawRw1yOG8iIlUKzbZl38kiTmwiQk8Xx/Yx6wyMYvHSjtHcNFD/c1rW/eQwY/aqFBvR3uwFHxo3efLVGnf06/BpLaZlyfkIfRY15m1/FHRcM8q66KT23kHrgj+2OcttT3LWtauWLWUzsRnWhnlGjyzgeQbKMPor+4mKErRtdLOrBoAaWB99GoQBmcnTGWuhBPj6rRw==",
			Signature: "-----BEGIN SSH SIGNATURE-----\nU1NIU0lHAAAAAQAAAhcAAAAHc3NoLXJzYQAAAAMBAAEAAAIBAODMOqw9v01F4y9g4e2tWP\n6xByBEG5taSAHfTgvfTe4HiqkVExIxECYwgdXRbwsWqXa02jlRShTL2+jMvtDQeqgusDqy\n4ns48fqyWnrEF+hU8RjLxNyK8OszhGLWuaWhcJcN/J1627LgA4LsQD1nEckMBH18Z2BMwm\n2iNNLan/Pe2jCQ1SPe0PMWdq0Yjr3mSXAK4HqMr7OxKQni6iFYLXUt3MqXZtNuqjZtDoRu\n7iRfCVor+NyOkZo+oUb0T0/61qtznOmHODNhksCQIoLMFgqyVh5XE+2yOriflCw2y0ElxN\nYL7jUjt7rnqwKpTIHvn1i95PXIE+h4I9SK56CZw2ApzADd3708FSgbE5dQb39YA4MlSWhp\n7RjUu6vLXbclijlTWPnG2vGnrE1+W3f8jeXu/x+zN0T/m9M1d4pGVVDW0F0Abvt6OAYaiO\nQzqXXhaZrX4eQXg1kRpO2gCfy0LKDHGv1sAHi0BzikMlyVrBHDXI4byIiVQrNtmXfySJOb\nCJCTxfH9jHrDIxi8dKO0dw0UP9zWtb95DBj9qoUG9He7AUfGjd58tUad/Tr8GktpmXJ+Qh\n9FjXmbX8UdFwzyrropPbeQeuCP7Y5y21Pcta1q5YtZTOxGdaGeUaPLOB5Bsow+iv7iYoSt\nG10s6sGgBpYH30ahAGZydMZa6EE+PqtHAAAABGZpbGUAAAAAAAAABnNoYTUxMgAAAhQAAA\nAMcnNhLXNoYTItNTEyAAACANrjoL5GcExI7owNilyimERsRSGDVCNtau/xVx0WIHwvIevj\nQDUKOYSIxs663WcThK+DzkK/TRZvlnckD3ef70t/UNXdk2PUC9Z874CBYx2jQ0aJLDjA2C\nscQL0PqOpVYJ4VsdPBJ5r1AX2/9W0NaIZnqRK3VTOL82EDHpnbTUTy1qXnpHWw8VfiMlax\nuCEjKn9jedzKRkyBeJv03+cEDOj4USsjwecaoNlGg33rdlafrp6hMFfGaOiJitPZ8M814w\nL+BL1bG45bAEYw85rDm6rWOjY/DQvlG6HS3G2r69qNEA4GplxHhKFPKz+WJ6NMsb+27371\nFivkPVe0TRVBpaB6s5rfPcQciBPq/rUtfwlaQHS2Ef+54c/0oRXA3674hkXZYwhgM1L+31\nzCpw7IZk/Ckkagxr3M69givZdfqLL7G502JsyPwdsfGj4YTrMw7vWp0uoesrmMoj2lZqvm\n77DZU8J839yqD2gS7ezXb4gluWIyooNcd9xl7nRuRTqs69slHYodJqnE9ELvA+lzo+251B\nTkwdW4EzXUAkhvms1zx4iOPLQPyu3rS2inklDbKbDXYed7lsqpq+Qs6AyTi27Nu+NI6SFj\nv7EVd1Nux3tXxgmZPcVfelpg93I/E4ZjDZE/vfSf8ykFEnvY+Q42KdJUvIx856g7BftxF3\nmBer5s\n-----END SSH SIGNATURE-----\n",
		},
		{
			Name:      "sk-ssh-ed25519@openssh.com",
			Key:       "sk-ssh-ed25519@openssh.com AAAAGnNrLXNzaC1lZDI1NTE5QG9wZW5zc2guY29tAAAAIGlgL27y/FLebK7nPpMmBrxpOCU9eIxyDhP6rq7DcdseAAAABHNzaDo=",
			Signature: "-----BEGIN SSH SIGNATURE-----\nU1NIU0lHAAAAAQAAAEoAAAAac2stc3NoLWVkMjU1MTlAb3BlbnNzaC5jb20AAAAgaWAvbv\nL8Ut5sruc+kyYGvGk4JT14jHIOE/qursNx2x4AAAAEc3NoOgAAAARmaWxlAAAAAAAAAAZz\naGE1MTIAAABnAAAAGnNrLXNzaC1lZDI1NTE5QG9wZW5zc2guY29tAAAAQHfNtXWSfVG02H\nyOyjo7fHV6kcY61Gq5nPblkmTzhDqYo5Rd9HRu5aId9YVhOzox024XLGUjxMHtYCiKjCkY\nLw8BAAAAIg==\n-----END SSH SIGNATURE-----\n",
		},
		{
			Name:      "sk-ecdsa-sha2-nistp256@openssh.com",
			Key:       "sk-ecdsa-sha2-nistp256@openssh.com AAAAInNrLWVjZHNhLXNoYTItbmlzdHAyNTZAb3BlbnNzaC5jb20AAAAIbmlzdHAyNTYAAABBBFjQymYQPdA4lRG+Yf/rLKxgm/AGqnN4wpb8w1X4W7s+3VMor8eB8wIJ3FubHRr2QKzlP56vaRnJpcA+/S3B7eQAAAAEc3NoOg==",
			Signature: "-----BEGIN SSH SIGNATURE-----\nU1NIU0lHAAAAAQAAAH8AAAAic2stZWNkc2Etc2hhMi1uaXN0cDI1NkBvcGVuc3NoLmNvbQ\nAAAAhuaXN0cDI1NgAAAEEEWNDKZhA90DiVEb5h/+ssrGCb8Aaqc3jClvzDVfhbuz7dUyiv\nx4HzAgncW5sdGvZArOU/nq9pGcmlwD79LcHt5AAAAARzc2g6AAAABGZpbGUAAAAAAAAABn\nNoYTUxMgAAAHgAAAAic2stZWNkc2Etc2hhMi1uaXN0cDI1NkBvcGVuc3NoLmNvbQAAAEkA\nAAAgeQgKbtxsvZVb1iPHWdMOt1pWFHbW6GvPSyJAzcRGaVcAAAAhAPRsy6THWGwYmusXcD\nPflbBeaqpzUfd/gaHjU/2To68UAQAAACY=\n-----END SSH SIGNATURE-----\n",
		},
		{
			Name:      "ssh-ed25519 sha256",
			Key:       "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIIAQv90+kSOSKZYlMoWO0eX6QZ1Nt5n2BviA4vFx3lgK",
			Signature: "-----BEGIN SSH SIGNATURE-----\nU1NIU0lHAAAAAQAAADMAAAALc3NoLWVkMjU1MTkAAAAggBC/3T6RI5IpliUyhY7R5fpBnU\n23mfYG+IDi8XHeWAoAAAAEZmlsZQAAAAAAAAAGc2hhMjU2AAAAUwAAAAtzc2gtZWQyNTUx\nOQAAAEAP8MvJBNUP/7rLtdoidM08d9WdTeXJFYhYh0A6RsP+Fp/opE1i2jrIK2goGNCFYv\n3SyZMXJcdhSgr70ixBgEMA\n-----END SSH SIGNATURE-----\n",
		},
	}

	for _, testCase := range testCases {
		err := verifySSHSignature(testCase.Key, testCase.Signature, data, namespace, true)
		if err != nil {
			t.Errorf("Verification failed for %s: %v", testCase.Name, err)
		}
	}

	sha256TestCase := testCases[5]
	err := verifySSHSignature(sha256TestCase.Key, sha256TestCase.Signature, data, namespace, false)
	if err == nil {
		t.Errorf("Verification succeeded for %s: expected failure", sha256TestCase.Name)
	}
}
