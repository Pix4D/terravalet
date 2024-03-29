#! /bin/sh
# DO NOT EDIT. Generated by terravalet.
# terravalet_output_format=2
#
# This script will move 5 items.

set -e

terraform state mv -lock=false -state=local.tfstate \
    'module.prometheus_instance.aws_instance.instance' \
    'module.prometheus.aws_instance.prometheus'

terraform state mv -lock=false -state=local.tfstate \
    'module.prometheus_instance.aws_route53_record.internal' \
    'aws_route53_record.private["prometheus"]'

terraform state mv -lock=false -state=local.tfstate \
    'module.prometheus_instance.aws_security_group_rule.extra_rules["reverseproxy_to_prometheus_pushprox"]' \
    'aws_security_group_rule.reverseproxy_to_prometheus_pushprox'

terraform state mv -lock=false -state=local.tfstate \
    'module.prometheus_instance.aws_volume_attachment.volumes["/dev/xvdh"]' \
    'module.prometheus.aws_volume_attachment.storage_attach'

terraform state mv -lock=false -state=local.tfstate \
    'module.prometheus_instance.null_resource.provision' \
    'module.prometheus.null_resource.provision_prometheus'

