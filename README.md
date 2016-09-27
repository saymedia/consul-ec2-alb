# consul-ec2-alb

`consul-ec2-alb` is a small daemon that syncs a healthy service list from
Consul into a target group in an EC2 Application Load Balancer (aka ELB2).

This utility can be used to automatically update one or more target lists
as nodes join and leave the cluster.

This tool is currently **experimental** and has not been used in production.
Adventurous people are welcome to give it a try, but please review the code
and test well before deploying in a production environment.

## Usage

Run `consul-ec2-alb` with a list of HCL configuration files on its command
line. The main element of the HCL config files is the `target_group` block,
which looks like this:

```hcl
target_group "arn:aws:elasticloadbalancing:us-west-2:12345:targetgroup/example/abc123" {
  service = "example-server"
}
```

The string in the header of the block is the ARN of the target group to sync
to, which can be found via the "Target Groups" panel in the EC2 console.

The `service` attribute is the name of the Consul service to monitor.

This block can optionally include a `datacenter` attribute, which specifies
which datacenter to monitor the service in. By default we will monitor whatever
datacenter the Consul agent belongs to.

### Consul Configuration

By default we connect to a local Consul agent on the standard port, with no
authentication token. This can be overridden using the optional `consul`
block:

```hcl
consul {
  address = "consul.example.net:443"
  scheme  = "https"
  token   = "abc123"
}
```

### AWS Configuration

By default we attempt to discover AWS credentials either in the environment
(via the standard `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` and
`AWS_SECURITY_TOKEN` variables) or by retrieving credentials from the EC2
metadata service.

Credentials can optionally be provided explicitly in configuration, using
the `aws` block:

```hcl
aws {
  access_key_id     = "abcdef1234567890"
  secret_access_key = "ssshhhsecret"

  # Optional; only used when using "assume role" tokens or STS delegation.
  security_token    = "xxxxxxxxxxx"
}
```

If no explicit configuration is provided and the environment variables are not
set, the program will attempt to reach the EC2 metadata service. If the
program is *not* running on an EC2 instance it is very likely that this call
will hang, making the program appear unresponsive. To fix this, explicitly
set credentials either via the environment or via the configuration file.

### High Availability

This program expects to be the only system populating the given target group,
though two concurrent instances reading from the same Consul service will
converge on the right outcome, albeit with some noise in the logs.

For a high-availability deployment it is strongly recommended to use
[the `consul lock` command](https://www.consul.io/docs/commands/lock.html)
to automatically choose an active node and keep other
nodes on standby.

Alternatively, deploy this program in a service scheduler like Nomad, to
ensure that it remains deployed on any one machine.

It's not yet known how well this application will scale by number of configured
target groups, but it should be possible to handle a modest amount of target
groups with a single active process, rather than e.g. running a separate
process for each load balancer, unless such a separation is convenient for
organizational or technical architecture reasons.

### Dynamic Target Group Configuration

This program directly monitors the consul healthcheck system to update the
membership of the target group, but it does *not* obtain the set of target
groups to monitor from Consul.

If dynamic configuration is desired, it's recommended to use
[`consul-template`](https://github.com/hashicorp/consul-template) (or
Nomad's built-in equivalent) to monitor a prefix within the Consul key/value
store, generate HCL configuration based on data stored there, and restart
`consul-ec2-alb` each time it changes.

It's assumed that changes to the set of synced target groups will be much
less frequent than changes to the members of those groups, and thus this
compromise avoids duplicating much of the functionality of `consul-template`.

## Contributing

As noted above, this program remains very experimental. If you give it a try
and find a bug or a missing feature, we'd be grateful of any contributions via
pull requests.

Here are some features that we are already thinking about:

* Ability to filter the monitored Consul services by tag, so that e.g. an ALB
  can include only the active in an active-standby setup, or other such
  Consul tagging use-cases.

* Ability to sync a service from multiple Consul datacenters into a single
  ALB target group. This would be useful in a deployment where, for example,
  Consul datacenter is mapped onto AWS availability zone but ALBs are deployed
  across many AZs within a single VPC.

* Possibility of using Consul's *prepared queries* feature for more elaborate
  selection and filtering of services.

* Optionally include unhealthy endpoints in the synced target group.
  ALB has its own healthcheck mechanism and so this could reduce the amount of
  churn in the target group membership in situations where service healthchecks
  are flapping.

Since this codebase is rather "early" it doesn't yet have any smooth dev
process or any automated tests. It's quite simple though, and it should be
possible to get started using the usual Go recipe:

* `go get github.com/saymedia/consul-ec2-alb`
* `go install github.com/saymedia/consul-ec2-alb`

## License

This program is distributed under the terms of the MIT license. For the
full license text, copyright information and warranty disclaimer please
see the separate file [LICENSE](LICENSE).
