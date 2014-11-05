package nameserver

import (
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func genForm(method string, url string, data url.Values) (resp *http.Response, err error) {
	req, err := http.NewRequest(method, url, strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return http.DefaultClient.Do(req)
}

func TestHttp(t *testing.T) {
	var (
		containerID     = "deadbeef"
		testDomain      = "weave.local."
		successTestName = "test1." + testDomain
		testAddr1       = "10.0.2.1/24"
		dockerIP        = "9.8.7.6"
	)

	var zone = new(ZoneDb)
	port := rand.Intn(10000) + 32768
	fmt.Println("Http test on port", port)
	go ListenHttp(testDomain, zone, port)

	time.Sleep(100 * time.Millisecond) // Allow for http server to get going

	// Ask the http server to add our test address into the database
	addrParts := strings.Split(testAddr1, "/")
	addrUrl := fmt.Sprintf("http://localhost:%d/name/%s/%s", port, containerID, addrParts[0])
	resp, err := genForm("PUT", addrUrl,
		url.Values{"fqdn": {successTestName}, "local_ip": {dockerIP}, "routing_prefix": {addrParts[1]}})
	assertNoErr(t, err)
	assertStatus(t, resp.StatusCode, http.StatusOK, "http response")

	// Check that the address is now there.
	ip, err := zone.MatchLocal(successTestName)
	assertNoErr(t, err)
	weaveIP, _, _ := net.ParseCIDR(testAddr1)
	if !ip.Equal(weaveIP) {
		t.Fatal("Unexpected result for", successTestName, ip)
	}

	// Adding exactly the same address should be OK
	resp, err = genForm("PUT", addrUrl,
		url.Values{"fqdn": {successTestName}, "local_ip": {dockerIP}, "routing_prefix": {addrParts[1]}})
	assertNoErr(t, err)
	assertStatus(t, resp.StatusCode, http.StatusOK, "http success response for duplicate add")

	// Now try adding the same address again with a different ident - should fail
	otherUrl := fmt.Sprintf("http://localhost:%d/name/%s/%s", port, "other", addrParts[0])
	resp, err = genForm("PUT", otherUrl,
		url.Values{"fqdn": {successTestName}, "local_ip": {dockerIP}, "routing_prefix": {addrParts[1]}})
	assertNoErr(t, err)
	assertStatus(t, resp.StatusCode, http.StatusConflict, "http response")

	// Delete the address
	resp, err = genForm("DELETE", addrUrl, nil)
	assertNoErr(t, err)
	assertStatus(t, resp.StatusCode, http.StatusOK, "http response")

	// Check that the address is not there now.
	_, err = zone.MatchLocal(successTestName)
	assertErrorType(t, err, (*LookupError)(nil), "nonexistent lookup")

	// Delete the address again, it should accept this
	resp, err = genForm("DELETE", addrUrl, nil)
	assertNoErr(t, err)
	assertStatus(t, resp.StatusCode, http.StatusOK, "http response")

	// Would like to shut down the http server at the end of this test
	// but it's complicated.
	// See https://groups.google.com/forum/#!topic/golang-nuts/vLHWa5sHnCE
}
