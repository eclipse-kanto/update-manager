[![Kanto logo](https://github.com/eclipse-kanto/kanto/raw/main/logo/kanto.svg)](https://eclipse.dev/kanto/)

# Eclipse Kanto - Update Manager

[![Coverage](https://github.com/eclipse-kanto/update-manager/wiki/coverage.svg)](#)

The update manager is a component responsible for the orchestration of OTA Updates towards a target device in a smarter way. The general idea is that the update manager receives a desired state specification for the whole target device and it is responsible to take the proper actions so that the current state of the target device matches the desired state.

The desired state comes in a descriptive way, a.k.a desired state manifest. The update manager is designed in such a way that it can easily be modified/extended to support various desired state representation formats.

The update manager works together with a number of additional components, named update agents and implementing a dedicated UpdateAgent API. These agents have the responsibility to update a certain domain inside the target device while the update manager is domain-neutral.

The update domains are, but not limited to:
- [Self-Update (OS Update) of connected Device](https://github.com/eclipse-leda/leda-contrib-self-update-agent)
- [Container Management](https://github.com/eclipse-kanto/container-management)
- IoT Device Firmware
- Embedded ECU Firmware

## Community

* [GitHub Issues](https://github.com/eclipse-kanto/update-manager/issues)
* [Mailing List](https://accounts.eclipse.org/mailing-list/kanto-dev)
