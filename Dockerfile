FROM alpine:3.21
RUN apk add --no-cache ca-certificates bash libvirt gcompat
COPY libvirt-cloud-controller-manager /bin/libvirt-cloud-controller-manager
ENTRYPOINT ["/bin/libvirt-cloud-controller-manager"]