whois-dns
=========

whois-dns 显示 Client 端出口的 DNS、HTTP 地址。

部署程序前需要将域名 NS 记录指向运行 whois-dns 程序的服务器地址。

```bash
$ curl http://localhost/ -v -L
* About to connect() to localhost port 80 (#0)
*   Trying 127.0.0.1... connected
* Connected to localhost (127.0.0.1) port 80 (#0)
> GET / HTTP/1.1
> User-Agent: curl/7.19.7 (x86_64-redhat-linux-gnu) libcurl/7.19.7 NSS/3.27.1 zlib/1.2.3 libidn/1.18 libssh2/1.4.2
> Host: localhost
> Accept: */*
>
< HTTP/1.1 302 Found
< Location: http://c84b09114301de3f6dd4ba075b83815d.localhost/feedback
< Date: Wed, 13 Dec 2017 05:28:10 GMT
< Content-Length: 81
< Content-Type: text/html; charset=utf-8
<
* Ignoring the response-body
* Connection #0 to host localhost left intact
* Issue another request to this URL: 'http://c84b09114301de3f6dd4ba075b83815d.localhost/feedback'
* About to connect() to c84b09114301de3f6dd4ba075b83815d.localhost port 80 (#1)
*   Trying 10.16.77.123... connected
* Connected to c84b09114301de3f6dd4ba075b83815d.localhost (10.16.77.123) port 80 (#1)
> GET /feedback HTTP/1.1
> User-Agent: curl/7.19.7 (x86_64-redhat-linux-gnu) libcurl/7.19.7 NSS/3.27.1 zlib/1.2.3 libidn/1.18 libssh2/1.4.2
> Host: c84b09114301de3f6dd4ba075b83815d.localhost
> Accept: */*
>
< HTTP/1.1 200 OK
< Date: Wed, 13 Dec 2017 05:28:10 GMT
< Content-Length: 68
< Content-Type: text/plain; charset=utf-8
<
* Connection #1 to host c84b09114301de3f6dd4ba075b83815d.localhost left intact
* Closing connection #0
* Closing connection #1
{"http_remote_addr":"127.0.0.1:32927","dns_remote_addr":"127.0.0.1"}
```