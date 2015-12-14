{% if pillar.get('network_provider', '').lower() == 'calico' %}

include:
  - docker

calicoctl:
  file.managed:
    - name: /usr/bin/calicoctl
    - source: https://github.com/projectcalico/calico-docker/releases/download/v0.10.0/calicoctl
    - source_hash: sha512=5dd8110cebfc00622d49adddcccda9d4906e6bca8a777297e6c0ffbcf0f7e40b42b0d6955f2e04b457b0919cb2d5ce39d2a3255d34e6ba36e8350f50319b3896
    - makedirs: True
    - mode: 744

calico-plugin:
  file.managed:
    - name: /usr/libexec/kubernetes/kubelet-plugins/net/exec/calico/calico
    - source: https://github.com/projectcalico/calico-kubernetes/releases/download/v0.7.0/calico_kubernetes
    - source_hash: sha512=f5764917522a0caec84b3ea61e191020b2e038cd6393b152e300739dbdc754e1c03bf1ba24f333d63e506cde8ed749f69a4b2e8717830c8c36625926167187b5
    - makedirs: True
    - mode: 744
    - require_in:
      - service: kubelet

plugin-config:
  file.managed:
    - name: /usr/libexec/kubernetes/kubelet-plugins/net/exec/calico/calico_kubernetes.ini
    - source: salt://calico/calico_kubernetes.ini
    - template: jinja
    - makedirs: True
    - mode: 744

calico-node:
  cmd.run:
    - name: calicoctl node
    - unless: docker ps | grep calico-node
    - env:
      - ETCD_AUTHORITY: "{{ grains.api_servers }}:6666"
    - require:
      - kmod: ip6_tables
      - kmod: xt_set
      - service: docker
      - file: calicoctl
      - file: plugin-config
      - file: calico-plugin

calico-ip-pool-reset:
  cmd.run:
    - name: calicoctl pool remove 192.168.0.0/16
    - onlyif: calicoctl pool show | grep 192.168.0.0/16
    - env:
      - ETCD_AUTHORITY: "{{ grains.api_servers }}:6666"
    - require:
      - service: docker
      - file: calicoctl
      - cmd: calico-node
    - require_in:
      - file: /usr/local/bin/kubelet

calico-ip-pool:
  cmd.run:
    - name: calicoctl pool add {{ grains['cbr-cidr'] }} --nat-outgoing
    - unless: calicoctl pool show | grep {{ grains['cbr-cidr'] }}
    - env:
      - ETCD_AUTHORITY: "{{ grains.api_servers }}:6666"
    - require:
      - service: docker
      - file: calicoctl
      - cmd: calico-node
    - require_in:
      - file: /usr/local/bin/kubelet

ip6_tables:
  kmod.present

xt_set:
  kmod.present

{% endif %}
