### Overview
The Update Manager interacts with the Update Agents using the Update Agent API. This API abstracts the specifics of individual domains and therefore introduces the possibility to implement new Update Agents as required. They can then be connected the to Update manager without the need for further implementation work. This brings the flexibility required to apply the onboard system to a wide range of usage scenarios.

### Message Format
The messages for the bidirectional exchange between the Update Manager and the Update Agents are carried in the following format:

```
{
  "activityId": "123e4567-e89b-12d3-a456-426614174000",
  "timestamp": 123456789,
  "payload": {} // actual message content as per message specification
}
```

### Message Data Model
The message data model has the following three metadata elements:

- `activityId` [string]: UUID generated by the backend which is used for correlating a set of device / backend messages with an activity entity (e.g. a desired state application process) on system level

- `timestamp` [int64]: Message creation timestamp. Number of milliseconds that have elapsed since the Unix epoch (00:00:00 UTC on 1 January 1970)

- `payload` [object]: Custom, unstructured message payload per message specification

### MQTT Topics
The Update Manager and each Update Agent bidirectionally exchange messages in the previously described format using MQTT. Each used topic reflects the respective Update Agent's `domain-identifier`. For usage in MQTT topics, this identifier is transformed to become the `mqtt-domain-identifier`:

- all dashes ("-") are removed from `domain-identifier`
- if `domain-identifier` does not end with "update", then "update" is appended.

Examples for `mqtt-domain-identifier`:

- containers -> containersupdate
- self-update -> selfupdate

| Topic | Direction | Purpose | Payload |
| - | - | - | - |
| `${mqtt_domain_identifier}/desiredstate` | UM -> UA | Informing the update agent for `${domain-identifier}` about a new desired state | Contains the desired state specification for the concrete domain |
| `${mqtt_domain_identifier}/desiredstate/command` | UM -> UA | Commanding the update agent for `${domain-identifier}` to perform an action | The command & baseline names are inside the payload |
| `${mqtt_domain_identifier}/desiredstatefeedback` | UA -> UM | Informing the Update Manager about the progress of a desired state application process within `${domain-identifier}` to perform an action | Contains the status of the update operation within the domain and a list of actions and their status |
| `${mqtt_domain_identifier}/currentstate` | UA -> UM | Reporting the current state of `${domain-identifier}` to the Update Manager | Current state representation of the domain with software/hardware nodes and associations between them. |
| `${mqtt_domain_identifier}/currentstate/get` | UM -> UA | Requesting a current state report from `${domain-identifier}` agent | Just a trigger, no payload. |

### Specialization of Update Agent API

Each update agent implementation shall come up with its own specific characteristics that need further documentation with the concrete Update Agent.
