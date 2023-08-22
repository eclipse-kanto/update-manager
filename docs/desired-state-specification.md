### Desired State Representation
A technology agnostic way of representing the target software state of the device is required to enable remote management of device software and configuration. The device software state model enables flexible representation of complex software topologies of a device as well as the related configuration. It also enables users and application developers to develop and deploy composite applications seamlessly based on device context changes.

### Desired State Data Model

The following table describes all supported properties and sections of the Desired State specification:

| Property | Type | Description |
| - | - | - |
| **General properties** | | | |
| baselines | JSON array | List of Desired State baselines |
| domains | JSON array | List of Desired State specifications for the domain agents |
| **Baseline properties** | | |
| title | string | Title of the baseline |
| description | string | Description of the baseline. A baseline is supposed to hold dependent components that are to be updated together. |
| preconditions | string | List of comma-separated precondition expressions |
| components | JSON array | Components of the baseline, each entry is prefixed with the domain name so that cross-domain dependecies can be handled too. |
| **Domain properties** | | |
| id | string | Identifier of the domain agent |
| config | JSON object | Set of domain-specific configuration properties as key/value pairs |
| components | JSON object | Set of components for the domain |
| **Config properties** | | |
| key | string | Key of the configuration property |
| value | string | Value of the configuration property |
| **Component properties** | | |
| id | string | Identifier of the component |
| version | string | Version of the component |
| config | JSON object | Set of component-specific runtime configuration properties as key/value pairs |

### Desired State Data Model Example

The following data structure is a holistic example of a device desired state:
```json
{
	"baselines": [
		{
			"title": "baseline-1",
			"description": "Bugfix because of problem 1",
			"preconditions": "AND(device.mode=MAINTENANCE,OR(device.battery>75,device.pluggedIn=true))",
			"components": [
				"custom-domain:app-1",
				"custom-domain:app-2"
			]
		},
		{
			"title": "composite-app123",
			"components": [
				"containers:xyz",
				"custom-domain:app-3",
				"custom-domain:app-4"
			]
		}
	],
	"domains": [
		{
			"id": "containers",
			"components": [
				{
					"id": "xyz",
					"version": "1.2.3",
					"config": [
						{
							"key": "image",
							"value": "container-registry.io/xyz:1.2.3"
						}
					]
				},
				{
					"id": "abc",
					"version": "4.5.6",
					"config": [
						{
							"key": "image",
							"value": "container-registry.io/abc:4.5.6"
						}
					]
				}
			]
		},
		{
			"id": "custom-domain",
			"components": [
				{
					"id": "app-1",
					"version": "1.0"
				},
				{
					"id": "app-2",
					"version": "4.3"
				},
				{
					"id": "app-3",
					"version": "342.444.195",
					"config": [
						{
							"key": "some.setting",
							"value": "abcd"
						}
					]
				},
				{
					"id": "app-4",
					"version": "568.484.195"
				}
			]
		},
		{
			"id": "self-update",
			"config": [
				{
					"key": "rebootRequired",
					"value": "true"
				}
			],
			"components": [
				{
					"id": "os-image",
					"version": "https://example.com/image.tar.gz"
				}
			]
		}
	]
}
```

### Configuration State Representation
Besides software components, the desired state representation needs to also support configuration state. This is used to provide runtime configuration to the components such as environment variables or secrets. In the state representation model, each software component has a set of configuration attached to it in a 1:1 relationship.

### Data Model and Domains
The purpose of this model is to describe the required data structure in a technology agnostic way.
The desired software state covers multiple domains of edge device, which can be extended as additional domains are exploited through newly developed update agents.
The device software state model is capable of supporting any domains. This is achieved by abstracting the domain specific installation technology via corresponding update agents.

### Dependencies between component updates
When transmitting a desired state to a device, the device needs to be informed about dependencies between the contained components e.g. a composite app can comprise of updates to components in different domains.
These requirements make it necessary to transmit such implicit dependencies from backend to the device. To achieve this flexibility, the dependencies are modeled by a root-level object `baselines` which is used to describe dependencies between elements from the domain's respective components sections. Each baseline represent a logical unit, which is comprised of set of components across multiple domains.
