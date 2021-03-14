# Firecracker toolbox

## CNI network

Prerequisites:
* [CNI plugins](https://github.com/containernetworking/plugins)
* [tc-redirect-tap](https://github.com/awslabs/tc-redirect-tap)

* custom CNI network

    ```sh
    cat <<EOF > /etc/cni/conf.d/50-firebox.conflist
    {
        "name": "firebox",
        "cniVersion": "0.4.0",
        "plugins": [
            {
                "type": "bridge",
                "name": "fireboxbr",
                "bridge": "fireboxbr0",
                "isDefaultGateway": true,
                "ipMasq": true,
                "hairpinMode": true,
                "ipam": {
                    "type": "host-local",
                    "subnet": "192.168.128.0/24",
                    "resolvConf": "/etc/resolv.conf"
                }
            },
            {
                "type": "firewall"
            },
            {
                "type": "tc-redirect-tap"
            }
        ]
    }
    EOF
    ``` 

## Build and import kernel and service image

Prerequisites:
* [ignite](https://github.com/weaveworks/ignite)

```sh
make clean vendor build docker-build docker-build-echo
make import-image import-kernel 

``` 
## Playground
### Run custom firectl 

```sh
make build && sudo bin/firebox firectl --jailer-enable --net-ns /var/run/netns/$(uuidgen)
``` 

### Run server

```sh
make build && sudo bin/firebox server --server-port 8080 --jailer-enable --net-ns /var/run/netns/$(uuidgen)
curl -X POST localhost:8080/vm/run
```

### Simple LB and probing to echo server


```sh
make build && sudo bin/firebox server --log-level=debug --server-port 8080 --jailer-enable --net-ns /var/run/netns/$(uuidgen)

curl -v -H 'Content-Type: application/json' -X POST http://localhost:8080/invoke -d '{"httpMethod": "GET"}'

curl -X POST localhost:8080/vm/run
curl -s -H 'Content-Type: application/json' -X POST http://localhost:8080/invoke -d '{"httpMethod": "POST", "rawPath": "/test/doit", "rawQueryString" : "key=val"}'
curl -X POST localhost:8080/vm/run
curl -s -H 'Content-Type: application/json' -X POST http://localhost:8080/invoke -d '{"httpMethod": "POST", "rawPath": "/test/doit", "rawQueryString" : "key=val"}' | jq -r '.body'  | base64 -d
...

```
