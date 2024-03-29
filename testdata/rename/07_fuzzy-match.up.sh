#! /bin/sh
# DO NOT EDIT. Generated by terravalet.
# terravalet_output_format=2
#
# This script will move 5 items.

set -e

terraform state mv -lock=false -state=local.tfstate \
    'aws_route53_record.private["prometheus"]' \
    'module.prometheus_instance.aws_route53_record.internal'

terraform state mv -lock=false -state=local.tfstate \
    'aws_security_group_rule.reverseproxy_to_prometheus_pushprox' \
    'module.prometheus_instance.aws_security_group_rule.extra_rules["reverseproxy_to_prometheus_pushprox"]'

terraform state mv -lock=false -state=local.tfstate \
    'module.prometheus.aws_instance.prometheus' \
    'module.prometheus_instance.aws_instance.instance'

terraform state mv -lock=false -state=local.tfstate \
    'module.prometheus.aws_volume_attachment.storage_attach' \
    'module.prometheus_instance.aws_volume_attachment.volumes["/dev/xvdh"]'

terraform state mv -lock=false -state=local.tfstate \
    'module.prometheus.null_resource.provision_prometheus' \
    'module.prometheus_instance.null_resource.provision'

