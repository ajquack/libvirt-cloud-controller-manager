libvirt-cloud-controller-manager Helm Chart

This Helm chart is the recommended installation method for libvirt-cloud-controller-manager.
Quickstart

First, install Helm 3.

The following snippet will deploy libvirt-cloud-controller-manager to the kube-system namespace.

# Sync the Libvirt Cloud helm chart repository to your local computer.
helm repo add lccm https://github.com/ajquack/
helm repo update lccm

# Install the latest version of the libvirt-cloud-controller-manager chart.
helm install lccm lccm/libvirt-cloud-controller-manager -n kube-system

Please note that additional configuration is necessary. See the main Deployment guide.

If you're unfamiliar with Helm it would behoove you to peep around the documentation. Perhaps start with the Quickstart Guide?
Upgrading from static manifests

If you previously installed libvirt-cloud-controller-manager with this command:

kubectl apply -f https://github.com/ajquack/libvirt-cloud-controller-manager/releases/latest/download/lccm.yaml

You can uninstall that same deployment, by running the following command:

kubectl delete -f https://github.com/ajquack/libvirt-cloud-controller-manager/releases/latest/download/lccm.yaml

Then you can follow the Quickstart installation steps above.
Configuration

This chart aims to be highly flexible. Please review the values.yaml for a full list of configuration options.

If you've already deployed lccm using the helm install command above, you can easily change configuration values:

helm upgrade lccm lccm/libvirt-cloud-controller-manager -n kube-system --set monitoring.podMonitor.enabled=true

Multiple replicas / DaemonSet

You can choose between different deployment options. By default the chart will deploy a single replica as a Deployment.

If you want to change the replica count you can adjust the value replicaCount inside the helm values. If you have more than 1 replica leader election will be turned on automatically.

If you want to deploy hccm as a DaemonSet you can set kind to DaemonSet inside the values. To adjust on which nodes the DaemonSet should be deployed you can use the nodeSelector and additionalTolerations values.
