<img src="resources/digitalkey.png" width="256" alt="nCryptAgent">

nCryptAgent
===========

*An SSH Agent for Hardware backed keys on Windows*

Ever been jealous of macOS users and their fancy Secure Enclave backed SSH Keys? Or wanted a nice GUI for managing keys like [Secretive](https://github.com/maxgoedjen/secretive)? nCryptAgent is your answer!

Use any smart card as an SSH key source, and manage them using a nice-ish GUI! Don't have a physical smart card or security key like a Yubikey? No problem -- Use the Microsoft Platform Crpyto Provider that is backed by your TPM for hardware backed keys!

Use your WebAuthN authenticator as your SSH key with `sk-ssh-ed25519@openssh.com` and `sk-ecdsa-sha2-nistp256@openssh.com` key types.

## Features
* Create TPM-backed hardware keys using the `Microsoft Platform Crypto Provider` (PCP)
* Create and use OpenSSH SK keys without middleware
  * `sk-ssh-ed25519@openssh.com` and `sk-ecdsa-sha2-nistp256@openssh.com` key types, along with their matching certificates
* Import and use nCrypt keys that are backed by Smart Cards
  * Yubikeys
  * [Virtual Smart Cards](https://learn.microsoft.com/en-us/windows/security/identity-protection/virtual-smart-cards/virtual-smart-card-overview)
  * ...any other smart card or PIV applet supported the `Microsoft Smart Card Key Storage Provider`
* A nice-ish GUI for managing your hardware-backed keys
* Supports multiple SSH Agent listeners:
  * OpenSSH for Windows
  * PuTTY/Pageant
  * WSL2
  * Cygwin/mSys/MinGW
* Notifications so you know when your key is being used
* Configurable PIN/Password cache, so you don't have to re-enter your PIN/Password for rapid successive key usage (not available for WebAuthN keys)
* OpenSSH Certificates
  * Adds support for OpenSSH certificates to PuTTY!

## Getting Started

* Download the latest release
* Click `Create Key` and enter a key name and container name.
  * Key name is a friendly descriptive name for the SSH Key
  * Container name is the nCrypt key container identifier which will be used. You can enter a memorable name such as `MY_KEY` or something random like a UUID.
* Select your Key Algorithm
* Enter a password or PIN
  * This can be empty if you wish to be a bit less secure
* Click save
* You now have a new SSH key, you can click the `Copy Key` button to copy the `authorized_keys` content to the clipboard and save it to the remote server. Alternatively you can copy the public key's path for use as a command line arg, or opening with another program.

You can use the key by configuring your SSH client to use nCryptAgent as its SSH agent. For OpenSSH for Windows and PuTTY this should work automatically, as long as those listeners are enabled in the `Config` tab. For WSL2 and Cygwin, you will need to set your `SSH_AUTH_SOCK` environment variable. The commands for doing this are available in the `Config` tab.

## Getting Started with WebAuthN Security Keys

* From the nCryptAgent main window, select the dropdown arrow in the bottom left and click on `Create new webauthn key`
* Enter a friendly name for the key and choose your key algorithm
* Click `Save` and you will be prompted to enter your pin and touch your security key
* Your key is now available for use

OpenSSH-SK keys require OpenSSH v8.4 or higher to use.

## Getting Started with Smart Cards

If you don't already have a certificate on your smart card, you'll need to create one.

### Yubikeys

* Open the YubiKey Manager App
* Select the PIV application
* Click on `Configure Certificates`
* Select an empty slot and click `Generate`
* Click through the wizard to create a self-signed certificate
* Once you have a certificate saved, follow the Use existing keys section below.

### TPM Smart Cards

* Create a card using the instructions below
* Use `certreq` and `certutil` to generate a self-signed certificate and install it to the smart card

Once you have a key added to nCryptAgent you can use it by configuring your SSH client to use nCryptAgent as its SSH agent. For OpenSSH for Windows and PuTTY this should work automatically, as long as those listeners are enabled in the `Config` tab. For WSL2 and Cygwin, you will need to set your `SSH_AUTH_SOCK` environment variable. The commands for doing this are available in the `Config` tab.

## Using existing Smart Card keys

If you have already generated a key on your smart card (for instance you have existing credentials on your Yubikey) you can import that key by clicking on the dropdown next to `Create Key` and selecting `Add existing nCrypt key`. Select your smart card reader from the dropdown, then select your existing key and enter a name. Click save and your existing key will be ready for use.

## OpenSSH Certificates

Since `ssh-add` does [not support adding certificates without a private key](https://bugzilla.mindrot.org/show_bug.cgi?id=3212), nCryptAgent checks for a matching certificate in its `PublicKeys` directory (`%AppData%\nCryptAgent\PublicKeys`). If you have an OpenSSH certificate you wish to use, you can either use the `Add Cert` button to attach a certificate to the currently selected key, or alternatively place the certificate in the `PublicKeys` directory with the correct name. The name format for certificates is `<MatchingCertificateFingerprint>-cert.pub`. 

For example, if an nCrypt key has a location of `%AppData%\nCryptAgent\PublicKeys\deadbeefd530ca2d01b3b74c8641fe29.pub` the matching certificate will be named `%AppData%\nCryptAgent\PublicKeys\deadbeefd530ca2d01b3b74c8641fe29-cert.pub`. 

## Creating a TPM-backed Smart Card

Users without a physical key can create a TPM-backed smart card:
* Ensure your TPM is enabled in BIOS or UEFI. Different manufacturers name the setting differently.
* Open a command prompt
* Run `tpmvscmgr create /name <Friendly_Name> /AdminKey DEFAULT /pin PROMPT /pinpolicy minlen 4 /generate` where `<Friendly_Name>` is a name you choose
* You can use `certreq` and `certutil` to load a certificate onto the smart card, after which you can `Add existing nCrypt Key` to import your Smart Card credentials into nCryptAgent

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

* Sometimes the PIN prompt does not obtain focus correctly and will pop up in the background.
* Sometimes the 'Copy' buttons do not correctly copy to clipboard.
