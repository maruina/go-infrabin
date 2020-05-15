# go-infrabin

[![Go Report Card](https://goreportcard.com/badge/github.com/maruina/go-infrabin)](https://goreportcard.com/report/github.com/maruina/go-infrabin)
[![Coverage Status](https://coveralls.io/repos/github/maruina/go-infrabin/badge.svg?branch=master)](https://coveralls.io/github/maruina/go-infrabin?branch=master)

[infrabin](https://github.com/maruina/infrabin) written in go.

**Warning: `go-infrabin` exposes sensitive endpoints and should NEVER be used on the public Internet.**

## Usage

`go-infrabin` exposes ports:

* `8888` as a service port
* `8899` as the admin port, for liveness and readiness probes
* `8053` TCP/UDP DNS Server port for testing dns clients

To override the default values:

* _INFRABIN_MAX_DELAY_ to change the maximum value for the `/delay` endpoint. Default to 120.

## Service Endpoints

* `GET /`
  * _returns_: a JSON response

    ```json
    {
        "hostname": "<hostname>",
        "kubernetes": {
            "pod_name": "<pod_name>",
            "namespace": "<namespace>",
            "pod_ip": "<pod_ip>",
            "node_name": "<node_name>"
        }
    }
    ```

* `GET /delay/<seconds>`
  * _returns_: a JSON response

    ```json
    {
        "delay": "<seconds>"
    }
    ```

* `GET /headers`
  * _returns_: a JSON response with [HTTP headers](https://pkg.go.dev/net/http?tab=doc#Header)

    ```json
    {
        "headers": "<request headers>"
    }
    ```

* `GET /env/<env_var>`
  * _returns_: a JSON response with the requested `<env_var>` or `404` if the environment variable does not exist

    ```json
    {
        "env": {
            "<env_var>": "<env_var_value>"
        }
    }
    ```

## Admin Endpoints

* `GET /liveness`
  * _returns_: a JSON response if healthy or the status code `503` if unhealthy.

    ```json
    {
        "liveness": "pass"
    }
    ```

## DNS Client Examples
```
 dig @localhost -p8053 cname.infrabin.com

; <<>> DiG 9.10.6 <<>> @localhost -p8053 cname.infrabin.com
; (2 servers found)
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 9832
;; flags: qr rd; QUERY: 1, ANSWER: 0, AUTHORITY: 1, ADDITIONAL: 0
;; WARNING: recursion requested but not available

;; QUESTION SECTION:
;cname.infrabin.com.		IN	A

;; AUTHORITY SECTION:
cname.infrabin.com.	300	IN	CNAME	infrabin.com.

;; Query time: 0 msec
;; SERVER: ::1#8053(::1)
;; WHEN: Fri May 15 16:13:38 BST 2020
;; MSG SIZE  rcvd: 80
```

```
dig @localhost -p8053 aaaarecord.infrabin.com

; <<>> DiG 9.10.6 <<>> @localhost -p8053 aaaarecord.infrabin.com
; (2 servers found)
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 13777
;; flags: qr rd; QUERY: 1, ANSWER: 1, AUTHORITY: 0, ADDITIONAL: 0
;; WARNING: recursion requested but not available

;; QUESTION SECTION:
;aaaarecord.infrabin.com.	IN	A

;; ANSWER SECTION:
aaaarecord.infrabin.com. 0	IN	AAAA	::1

;; Query time: 0 msec
;; SERVER: ::1#8053(::1)
;; WHEN: Fri May 15 16:14:06 BST 2020
;; MSG SIZE  rcvd: 92
```

## Errors

* `400`
  * _returns_:

  ```json
    {
        "error": "<reason>"
    }
  ```
