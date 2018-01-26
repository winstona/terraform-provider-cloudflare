Terraform Provider
==================

- Website: https://www.terraform.io
- [![Gitter chat](https://badges.gitter.im/hashicorp-terraform/Lobby.png)](https://gitter.im/hashicorp-terraform/Lobby)
- Mailing list: [Google Groups](http://groups.google.com/group/terraform-tool)

<img src="https://cdn.rawgit.com/hashicorp/terraform-website/master/content/source/assets/images/logo-hashicorp.svg" width="600px">

Requirements
------------

-	[Terraform](https://www.terraform.io/downloads.html) 0.10.x
-	[Go](https://golang.org/doc/install) 1.8 (to build the provider plugin)

Building The Provider
---------------------

Clone repository to: `$GOPATH/src/github.com/terraform-providers/terraform-provider-$PROVIDER_NAME`

```sh
$ mkdir -p $GOPATH/src/github.com/terraform-providers; cd $GOPATH/src/github.com/terraform-providers
$ git clone git@github.com:terraform-providers/terraform-provider-$PROVIDER_NAME
```

Enter the provider directory and build the provider

```sh
$ cd $GOPATH/src/github.com/terraform-providers/terraform-provider-$PROVIDER_NAME
$ make build
```

Using the provider
----------------------

Example DNS records:

```
resource "cloudflare_record" "cf_record" {
  domain = "example.com"
  name   = "subdomain"
  value  = "otherhost.example.com"
  type   = "CNAME"
  ttl    = "3600"
}

resource "cloudflare_record" "mx_cf_record" {
  domain = "example.com"
  name   = "example.com"
  value  = "mxhost.example.com"
  priority = "10"
  type   = "MX"
  ttl    = "3600"
}

```

Format for import key: "<domain>/<fqdn>/<record_type>" (ex. "example.com/subdomain.example.com/CNAME")

MX records are handled slightly differently, as they require an additional record index: "<domain>/<fqdn>/MX/<record_index>"

Import command example:

```
terraform import cloudflare_record.cf_record "example.com/subdomain.example.com/A"
terraform import cloudflare_record.mx_cf_record[0] "example.com/example.com/MX/0"
terraform import cloudflare_record.mx_cf_record[1] "example.com/example.com/MX/1"
```


Developing the Provider
---------------------------

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (version 1.8+ is *required*). You'll also need to correctly setup a [GOPATH](http://golang.org/doc/code.html#GOPATH), as well as adding `$GOPATH/bin` to your `$PATH`.

To compile the provider, run `make build`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

```sh
$ make bin
...
$ $GOPATH/bin/terraform-provider-$PROVIDER_NAME
...
```

In order to test the provider, you can simply run `make test`.

```sh
$ make test
```

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```sh
$ make testacc
```
