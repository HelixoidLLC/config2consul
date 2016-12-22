resource "tls_private_key" "ca" {
  algorithm = "RSA"
  rsa_bits = 2048
}

resource "tls_self_signed_cert" "ca" {
  key_algorithm = "${tls_private_key.ca.algorithm}"
  private_key_pem = "${tls_private_key.ca.private_key_pem}"
  validity_period_hours = 43800
  is_ca_certificate = "true"
  allowed_uses = ["cert_signing", "key_encipherment", "client_auth", "server_auth"]
  subject = {
    common_name = "*.${var.consul_domain}"
    organization = "${var.organization_name}"
    organizational_unit = "${var.organization_unit}"
    street_address = ["${var.organization_street}"]
    locality = "${var.organization_locality}"
    province = "${var.organization_province}"
    country = "${var.organization_country}"
    postal_code = "${var.organization_zip}"
    serial_number = "1"
  }
}

resource "tls_private_key" "consul_server" {
  count = "${var.consul_instance_count}"
  algorithm = "RSA"
  rsa_bits = 2048
}

resource "tls_cert_request" "consul_server" {
  count = "${var.consul_instance_count}"
  key_algorithm = "${element(tls_private_key.consul_server.*.algorithm, count.index)}"
  private_key_pem = "${element(tls_private_key.consul_server.*.private_key_pem, count.index)}"
  subject = {
    common_name = "server.${var.consul_data_center[0]}.${var.consul_domain}"
    organization = "${var.organization_name}"
    organizational_unit = "${var.organization_unit}"
    street_address = ["${var.organization_street}"]
    locality = "${var.organization_locality}"
    province = "${var.organization_province}"
    country = "${var.organization_country}"
    postal_code = "${var.organization_zip}"
    serial_number = "1"
  }
  dns_names = [
    "${format("consul%02d", count.index)}.${var.consul_domain}",
    "server.${var.consul_data_center[0]}.${var.consul_domain}"
  ]
}

resource "tls_locally_signed_cert" "consul_server" {
  count = "${var.consul_instance_count}"
  cert_request_pem = "${element(tls_cert_request.consul_server.*.cert_request_pem, count.index)}"
  ca_key_algorithm = "${tls_private_key.ca.algorithm}"
  ca_private_key_pem = "${tls_private_key.ca.private_key_pem}"
  ca_cert_pem = "${tls_self_signed_cert.ca.cert_pem}"
  allowed_uses = ["key_encipherment", "digital_signature", "server_auth", "client_auth"]
  validity_period_hours = 43800
}

resource "tls_private_key" "consul_client" {
  algorithm = "RSA"
  rsa_bits = 2048
}

resource "tls_cert_request" "consul_client" {
  key_algorithm = "${tls_private_key.consul_client.algorithm}"
  private_key_pem = "${tls_private_key.consul_client.private_key_pem}"
  subject = {
    common_name = "*.${var.consul_domain}"
    organization = "${var.organization_name}"
    organizational_unit = "${var.organization_unit}"
    street_address = ["${var.organization_street}"]
    locality = "${var.organization_locality}"
    province = "${var.organization_province}"
    country = "${var.organization_country}"
    postal_code = "${var.organization_zip}"
    serial_number = "2"
  }
  dns_names = [
    "*.${var.consul_domain}",
    "*.*.${var.consul_domain}",
    "*.*.*.${var.consul_domain}",
    "*.*.*.*.${var.consul_domain}"
  ]
}

resource "tls_locally_signed_cert" "consul_client" {
  cert_request_pem = "${tls_cert_request.consul_client.cert_request_pem}"
  ca_key_algorithm = "${tls_private_key.ca.algorithm}"
  ca_private_key_pem = "${tls_private_key.ca.private_key_pem}"
  ca_cert_pem = "${tls_self_signed_cert.ca.cert_pem}"
  allowed_uses = ["key_encipherment", "digital_signature", "client_auth"]
  validity_period_hours = 43800
}

############################
# Dump output

resource "null_resource" "dump" {
  provisioner "local-exec" {
    command = "mkdir -p ${var.out_path}"
  }
  provisioner "local-exec" {
    command = "echo \"${tls_private_key.ca.private_key_pem}\" > ${var.out_path}/ca.key"
  }
  provisioner "local-exec" {
    command = "echo \"${tls_self_signed_cert.ca.cert_pem}\" > ${var.out_path}/ca.crt"
  }
}

resource "null_resource" "dump_consul_client" {
  provisioner "local-exec" {
    command = "echo \"${tls_private_key.consul_client.private_key_pem}\" > ${var.out_path}/consul_client.key"
  }

  provisioner "local-exec" {
    command = "echo \"${tls_locally_signed_cert.consul_client.cert_pem}\" > ${var.out_path}/consul_client.crt"
  }
}

resource "null_resource" "dump_consul_server" {
  count = "${var.consul_instance_count}"

  provisioner "local-exec" {
    command = "echo \"${element(tls_private_key.consul_server.*.private_key_pem, count.index)}\" > ${var.out_path}/consul_server_${format("%02d", count.index)}.key"
  }

  provisioner "local-exec" {
    command = "echo \"${element(tls_locally_signed_cert.consul_server.*.cert_pem, count.index)}\" > ${var.out_path}/consul_server_${format("%02d", count.index)}.crt"
  }
}
