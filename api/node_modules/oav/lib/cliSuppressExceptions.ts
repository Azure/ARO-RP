// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

import { log } from "./util/logging";

export const cliSuppressExceptions = async (f: () => Promise<number>): Promise<void> => {
  try {
    process.exitCode = await f();
  } catch (err) {
    const message = `fatal error: ${(err as any).message}, ${JSON.stringify(err)}`;
    log.error(message);
    // eslint-disable-next-line no-console
    console.error(message);
    process.exitCode = 1;
  }
};
