prom-config-api
===============

This is a tiny API that allows clients to remotely list, add, and remove hosts on a Prometheus server using [file-based service discovery](https://prometheus.io/docs/operating/configuration/#<file_sd_config>).

Defaults that can be overriden with environment variables:

* `LISTEN=:9003`
* `PROMDIR=/opt/prometheus`

Routes:

* `GET /hosts`
* `POST /hosts`
* `DELETE /hosts/{alias}`

Data to `POST /hosts`:

```js
{
   "Address": "10.0.0.5",
   "Alias": "prod-db01"
}
```

