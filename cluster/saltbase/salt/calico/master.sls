{% if pillar.get('network_provider', '').lower() == 'calico' %}

calicoctl:
  file.managed:
    - name: /usr/bin/calicoctl
    - source: https://github.com/projectcalico/calico-docker/releases/download/v0.12.0/calicoctl
    - source_hash: sha512=001754f9a7ccbd434356c02ec30017448823ddcc35cda394b67680e67bda8cae704467863ca84944a940410886eba5e500f005f4744ea01204c56afc8ff12990
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
      - cmd: etcd

etcd:
  cmd.run:
    - unless: docker ps | grep calico-etcd
    - name: >
               docker run --name calico-etcd -d --restart=always -p 6666:6666
               -v /varetcd:/var/etcd
               gcr.io/google_containers/etcd:2.0.8
               /usr/local/bin/etcd --name calico
               --data-dir /var/etcd/calico-data
               --advertise-client-urls http://{{grains.api_servers}}:6666
               --listen-client-urls http://0.0.0.0:6666
               --listen-peer-urls http://0.0.0.0:2380
               --initial-advertise-peer-urls http://{{grains.api_servers}}:2380
               --initial-cluster calico=http://{{grains.api_servers}}:2380

ip6_tables:
  kmod.present

xt_set:
  kmod.present

{% endif %}