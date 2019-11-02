---
menu:
    main:
        parent: "How-Tos"
        weight: 26
title: "Validators"
date: 2019-07-28T17:24:32-07:00
---
Validators allow you to run checks on the probe request output (if any). For
example, you can specify if you expect the probe output to match a certain
regex or return a certain status code (for HTTP). You can configure more than
one validators and all validators should succeed for the probe to be marked as
success.

{{< highlight bash >}}
probe {
    name: "google_homepage"
    type: HTTP
    targets {
        host_names: "www.google.com"
    }
    interval_msec: 10000    # Probe every 10s

    # This validator should succeed.
    validator {
        name: "status_code_2xx"
        http_validator {
            success_status_codes: "200-299"
        }
    }

    # This validator will fail, notice missing 'o' in our regex.
    validator {
        name: "gogle_re"
        regex: "gogle"
    }
}
{{< / highlight >}}

(Full listing: https://github.com/google/cloudprober/blob/master/examples/validators/cloudprober_validator.cfg)

To make the debugging easier, validation failures are logged and exported as an
independent map counter -- *validation_failure*, with *validator* key. For
example, the above example will result in the following counters being
exported after 5 runs:

{{< highlight bash >}}
total{probe="google_homepage",dst="www.google.com"} 5
success{probe="google_homepage",dst="www.google.com"} 0
validation_failure{validator="status_code_2xx",probe="google_homepage",dst="www.google.com"} 0
validation_failure{validator="gogle_re",probe="google_homepage",dst="www.google.com"} 5
{{< / highlight >}}

Note that validator counter will **not** go up if probe fails for other
reasons, for example web server timing out. That's why you typically don't want
to alert only on validation failures. That said, in some cases, validation
failures could be the only thing you're interested in, for example, if you're
trying to make sure that a certain copyright is always present in your web
pages or you want to catch data integrity issues in your network.

Let's take a look at the types of validators you can configure.

## Regex Validator

Regex validator simply checks for a regex in the probe request output. It works
for all probe types except for UDP and UDP_LISTENER - these probe types don't
support any validators at the moment.

## HTTP Validator

HTTP response validator works only for the HTTP probe type. You can currently
use HTTP  validator to define success and failure status codes (represented by
success_status_codes and failure_stauts_codes in the config):

* If *failure_status_codes* is defined and response status code falls within
 that range, validator is considered to have failed.
* If *success_status_codes* is defined and response status code *does not*
 fall within that range, validator is considered to have failed.

We can possibly add more checks to HTTP validator in the future, for example to
verify headers.

## Data Integrity Validator

Data integrity validator is designed to catch the packet corruption issues in
the network. We have a basic check that verifies that the probe output is made
up purely of a pattern repeated many times over.

