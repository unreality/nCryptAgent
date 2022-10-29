<img src="resources/digitalkey.png" width="200">

nCryptAgent
===========

*An SSH Agent for Hardware backed keys on Windows*

Ever been jealous of MacOS users and their fancy Secure Enclave backed SSH Keys? Or wanted a nice GUI for managing keys like [Secretive](https://github.com/maxgoedjen/secretive)? nCryptAgent is your answer!

Use any smart card as an SSH key source, and manage them using a nice-ish GUI! Don't have a physical smart card or security key like a Yubikey? No problem -- Create a Virtual Smart Card that is backed by your TPM for hardware backed keys!

## Features
* Import or Create nCrypt keys that are backed by hardware
  * Windows TPM Smart Card
  * Yubikeys
  * Other smart cards
* An acceptable GUI for managing your hardware-backed keys
* Supports multiple SSH Agent listeners:
  * OpenSSH for Windows
  * PuTTY/Pageant
  * WSL2
  * Cygwin/mSys/MinGW
* Notifications so you know when your key is being used
* Configurable PIN cache so you dont have to re-enter your PIN for rapid successive key usage
* OpenSSH Certificates
  * Adds support for OpenSSH certificates to PuTTY!

## Getting Started

* Download the latest release
* Click `Create Key` and enter a key name and container name. Container name is the nCrypt key container identifier which will be used. You can enter a memorable name such as `MY_KEY` or something random like a UUID.
* Select your Key Algorithm
  * Windows TPM Smart Cards generally only support `RSA-2048`
  * Yubikeys support `RSA-2048` and `ECDSA-256`, and probably some others
* Click save
  * You may be asked to select which smart card to use to complete the action
  * If you have chosen a key algorithm the smart card does not support, or an invalid container name you will be presented with an error
* Enter your smart card PIN
* You now have a new SSH key, you can click the `Copy Key` button to copy the `authorized_keys` content to the clipboard and save it to the remote server. Alternatively you can copy the public key's path for use as a command line arg, or opening with another program.

You can use the key by configuring your SSH client to use nCryptAgent as its SSH agent. For OpenSSH for Windows and PuTTY this should work automatically, as long as those listeners are enabled in the `Config` tab. For WSL2 and Cygwin, you will need to set your `SSH_AUTH_SOCK` environment variable. The commands for doing this are available in the `Config` tab.

## Using existing keys

If you have already generated a key on your smart card (for instance you have existing credentials on your Yubikey) you can import that key by clicking on the dropdown next to `Create Key` and selecting `Add existing nCrypt key`. Select your smart card reader from the dropdown, then select your existing key and enter a name. Click save and your existing key will be ready for use.

## OpenSSH Certificates

Since `ssh-add` does [not support adding certificates without a private key](https://bugzilla.mindrot.org/show_bug.cgi?id=3212), nCryptAgent checks for a matching certificate in its `PublicKeys` directory (`%AppData%\nCryptAgent\PublicKeys`). If you have an OpenSSH certificate you wish to use, you can either use the `Add Cert` button to attach a certificate to the currently selected key, or alternatively place the certificate in the `PublicKeys` directory with the correct name. The name format for certificates is `<MatchingCertificateFingerprint>-cert.pub`. 

For example, if an nCrypt key has a location of `%AppData%\nCryptAgent\PublicKeys\deadbeefd530ca2d01b3b74c8641fe29.pub` the matching certificate will be named `%AppData%\nCryptAgent\PublicKeys\deadbeefd530ca2d01b3b74c8641fe29-cert.pub`. 

## Creating a TPM-backed Smart Card

Users without a physical key can create a TPM-backed smart card:
* Ensure your TPM is enabled in BIOS or UEFI. Different manufacturers name the setting differently.
* Open a command prompt
* Run `tpmvscmgr create /name <Friendly_Name> /AdminKey DEFAULT /pin PROMPT /pinpolicy minlen 4 /generate` where `<Friendly_Name>` is a name you choose
* You should now be able to add new keys in nCryptAgent

You can delete your TPM smart card with:
* `tpmvscmgr.exe destroy /instance <DeviceID>` where `<DeviceID>` is the id of the tpm smart card. If you only have one tpm smart card, this will be `ROOT\SMARTCARDREADER\0000`
* To get a list of `DeviceIDs` run `wmic path win32_PnPEntity where "DeviceID like '%smartcardreader%'" get DeviceID,Name,Status`

## Building

* To build you'll need `windres` which can be obtained by downloading the latest release of [llvm-mingw](https://github.com/mstorsjo/llvm-mingw)
* Download go deps with `go mod tidy`
* `windres.exe -i resources.rc -o rsrc.syso -O coff`
* `go build -ldflags "-H=windowsgui" -o build\nCryptAgent.exe`

I'll get around to making a proper build script at some point...

## Known Issues

* Sometimes the PIN prompt does not obtain focus correctly and will popup in the background.
* Sometimes the 'Copy' buttons do not correctly copy to clipboard.