// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

import * as yaml from "js-yaml"

export class ReadMeBuilder {
    public readonly getVersionDefinition = (yamlBody: any, tag: string) => `
### Tag: ${tag}

These settings apply only when \`--tag=${tag}\` is specified on the command line.

\`\`\`yaml $(tag) == '${tag}'
${yaml.dump(yamlBody, { lineWidth: -1 })}\`\`\`
`;

    public getSuppressionSection = () => `
## Suppression

\`\`\`yaml
${yaml.dump({ directive: [] }, { lineWidth: -1 })}\`\`\`
`;

}
