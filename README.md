# http2curl

:triangular_ruler: Convert Golang's http.Request to CURL command line

To do the reverse operation, check out [mholt/curl-to-go](https://github.com/mholt/curl-to-go).

## Example

```go
import (
    "http"
    "github.com/chodges15/http2curl"
)

data := bytes.NewBufferString(`{"hello":"world","answer":42}`)
req, _ := http.NewRequest("PUT", "http://www.example.com/abc/def.ghi?jlk=mno&pqr=stu", data)
req.Header.Set("Content-Type", "application/json")

command, _ := http2curl.GetCurlCommand(req)
fmt.Println(command)
// Output: curl -X PUT -d '{"hello":"world","answer":42}' -H 'Content-Type: application/json' 'http://www.example.com/abc/def.ghi?jlk=mno&pqr=stu' 
```


With gzip decompression enabled:
```go
import (
"http"
"github.com/chodges15/http2curl"
)

func compressData(data []byte) []byte {
    var buf bytes.Buffer
    gz := gzip.NewWriter(&buf)
    gz.Write(data)
    gz.Close()
    return buf.Bytes()
}


body := compressData([]byte(`{"test":"gzip"}`))
req, _ := http.NewRequest("POST", "http://example.com", bytes.NewReader(body))
req.Header.Set("Content-Encoding", "gzip")

command, _ := http2curl.GetCurlCommand(req, WithAutoDecompressGZIP())
fmt.Println(command)
// Output: curl -X 'POST' -d '{"test":"gzip"}' 'http://example.com'
```

## Install

```bash
go get github.com/chodges15/http2curl
```

## Usages

- https://github.com/parnurzeal/gorequest
- https://github.com/scaleway/scaleway-cli
- https://github.com/nmonterroso/cowsay-slackapp
- https://github.com/moul/as-a-service
- https://github.com/gavv/httpexpect
- https://github.com/smallnest/goreq

## License

Based on moul.io/http2curl/v2
© 2019-2021 [Manfred Touron](https://manfred.life)

Licensed under the [Apache License, Version 2.0](https://www.apache.org/licenses/LICENSE-2.0) ([`LICENSE-APACHE`](LICENSE-APACHE)) or the [MIT license](https://opensource.org/licenses/MIT) ([`LICENSE-MIT`](LICENSE-MIT)), at your option. See the [`COPYRIGHT`](COPYRIGHT) file for more details.

`SPDX-License-Identifier: (Apache-2.0 OR MIT)`
