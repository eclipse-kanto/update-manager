### Owner Consent
Sometimes an update comes with a down time or legal obligations and the device owner should be asked for a consent. For this case the Update Manager(UM) has the ability to request the owner's consent at single or multiple phases during the update process. When the owner approves the update through a Human Machine Interface(HMI), the UM continues to orchestrate it across the Update Agents(UAs). However, when the update gets denied the UM commands the UAs to rollback to the state before the update.

### Owner Consent Data Model

The following table describes all supported properties and sections of the Owner Consent specification:

| Property | Type | Description |
| - | - | - |
| **Consent properties** | | |
| command | string | [Command UM is about to issue to the UAs, for which an owner's consent is needed ](#supported-owner-consent-commands) |
| **Consent Feedback properties** | | |
| status | string | [Status of the consent feedback](#supported-owner-consent-statuses) |

### Supported Owner Consent commands

The list of the supported UM->UA commands for which the owner's approval could be requested :

| Status | Description |
| - | - |
| DOWNLOAD | Denotes that the owner consent is requested before download phase |
| UPDATE | Denotes that the owner consent is requested before the update phase |
| ACTIVATE | Denotes that the owner consent is requested before the activation phase |

### Supported Owner Consent statuses

The list of the supported feedback statuses :

| Status | Description |
| - | - |
| APPROVED | Denotes that the owner approved the update operation |
| DENIED | Denotes that the owner has denied the update operation |

### Owner Consent Data Model Example

This is a full example of a owner consent message for a device that is currently applying a desired state. In this example, the owner consent is done before the `DOWNLOAD` command, the UM needs consent to instruct the UAs to download the needed artifacts for the update:
```json
{
	"command": "DOWNLOAD"
}
```

### Owner Consent Feedback Data Model Example

This is a full example of a owner consent feedback message for a device that is currently applying a desired state. In this example, the owner has approved the update and the Update Manager will proceed to orchestrate the update on the device:
```json
{
	"status": "APPROVED"
}
```