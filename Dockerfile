FROM scratch

ADD ca-certificates.crt /etc/ssl/certs/
ADD k8s-bitflow /

CMD ["/k8s-bitflow", "operator"]