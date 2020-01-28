# Prepare your development environment using MacOS X

We are open to developers on OSX working on this repository. We are asking
 macOS users to setup GNU utils on their machines.

We are aiming to limit the amount of shell scripting, etc. in the repository,
installing the GNU utils on OSX will minimise the chances of unexpected
differences in command line flags, usages, etc., and make it easier for
everyone to ensure compatibility down the line.

## Guidances

```bash
# GNU Utils
brew install coreutils
brew install findutils

# Install envsubst
brew install gettext
brew link --force gettext

# Install
brew install gpgme

# GNU utils
# Ref: https://web.archive.org/web/20190704110904/https://www.topbug.net/blog/2013/04/14/install-and-use-gnu-command-line-tools-in-mac-os-x
# gawk, diffutils, gzip, screen, watch, git, rsync, wdiff
export PATH="/usr/local/bin:$PATH"
# coreutils
export PATH="/usr/local/opt/coreutils/libexec/gnubin:$PATH"
# findutils
export PATH="/usr/local/opt/findutils/libexec/gnubin:$PATH"
```
