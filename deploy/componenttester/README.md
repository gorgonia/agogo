# agogo component tester!

## requirements:

- bash 4+
- GitHub user/pass
- a $HOME/.aws/credentials file configured with your API key/secret (generated from the console)
- your own keypair (if you want to connect to the instance for manual ops)

Unfortunately OSX defaults to bash 3.2.57(1)-release (lame)

To install with Homebrew:
```sh
brew install bash
```
This will place a link in /usr/local/bin/bash

If you've got some other fancy setup, just edit componenttester.sh and update the shebang line

Finally, if you want to connect to the instance, you will need your own keypair.

## how to run:
```sh
./componenttester.sh
```
Follow the instructions.

Be sure not to leave any stacks running (the script tries to be clever about this)

## todo:
- spit out IP of instance if user keeps stack
- capture user's IP for inbound SSH
