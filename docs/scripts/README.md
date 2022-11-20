## Adapting Drive with JavaScript

First create a new js file where all the code for Drive is located.

This script will run in the otto environment, which is only **ES5** compatible, it **does not** support the DOM API, and only support limited APIs, which you can find in [`global.d.ts`](https://github.com/devld/go-drive/blob/master/docs/scripts/global.d.ts) and [`drive.d.ts`](https://github.com/devld/go-drive/blob/master/docs/scripts/env/drive.d.ts).

In this file, the first line is the display name of the Drive, and from the second line to the next empty line is the description(markdown supported) of the Drive.

Next, you can write the Drive's code.

There are three main methods `defineCreate`, `defineInitConfig` and `defineInit`.

- `defineCreate`: Creates an instance of the Drive
- `defineInitConfig`: Get the configuration form for each step of the Drive
- `defineInit`: the user fills the form returned by `defineInitConfig`, and the data will be passed to this method, you need to do some initialization in this method

This whole script is evaluated when getting the initialization configuration (`initConfig`), initializing (`init`) or creating (`create`), and will not be executed again when the Drive is created.

You can see examples of existing implementations in [`script-drives`](https://github.com/devld/go-drive/tree/master/script-drives).


Once you finish all, then you can copy this js file to `/script-drives`.
