# P1-06d34: distinguish pre-existing host forwarding conditions

Parent e87e47b0e4d95f520efabd7696c3e734b363f06f.
Earlier Hosted job 140222397450: host TLS PASS, NAT=true,
IPv4 forwarding=1, FORWARD DROP 0->25, source TCP/443 conntrack
zero, four guest curl28/HTTP000. Original independent real Box WAN
PASS at 92ffeda retained (D14). Read-only observation
020b2e88c3dee32540b74877bb56cc08 exit zero in the dedicated WAN
Box: docker inactive, Incus/BoxWeave active, forwarding=1,
native NAT=true, prior owned guest running, iptables unavailable.
This is a current environment comparison, not a new WAN positive.

Incus upstream firewall documentation
https://linuxcontainers.org/incus/docs/main/howto/network_bridge_firewalld/
identifies a possible Docker/Incus forwarding policy conflict.
This is a hypothesis, not confirmed attribution in Hosted.
Add three read-only aggregate observations (prebootstrap, before,
after) in the existing exact installed Scoped S2 mode: Docker service
status, FORWARD policy/jump presence, IPv4/inet forward-chain policy
counts. No packet contents, raw host rules, firewall mutation,
new destination, product bypass, simulated replacement or PASS
inference. Parser calibration and shell/matrix verification required.
Owned Box 9d374e18-196a-493d-bf1f-2d582589f04d
/root/pr191-p1-d34. Edits c69f0dee1b0ef353bd63664a260d3ae8,
e205d24c89af6628a834bf4829f5d437. Exact staged verification and
publication evidence in PR. Hosted positive and final native gate OPEN.
