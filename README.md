<img src="resources/digitalkey.png" width="256" alt="nCryptAgent">

nCryptAgent
===========

*An SSH Agent for Hardware backed keys on Windows*

Ever been jealous of macOS users and their fancy Secure Enclave backed SSH Keys? Or wanted a nice GUI for managing keys like [Secretive](https://github.com/maxgoedjen/secretive)? nCryptAgent is your answer!

Use any smart card as an SSH key source, and manage them using a nice-ish GUI! Don't have a physical smart card or security key like a Yubikey? No problem -- Use the Microsoft Platform Crypto Provider that is backed by your TPM for hardware backed keys!

Use your WebAuthN authenticator as your SSH key with `sk-ssh-ed25519@openssh.com` and `sk-ecdsa-sha2-nistp256@openssh.com` key types.

## Features
* Create TPM-backed hardware keys using the `Microsoft Platform Crypto Provider` (PCP)
* Create and use OpenSSH SK keys without middleware
  * `sk-ssh-ed25519@openssh.com` and `sk-ecdsa-sha2-nistp256@openssh.com` key types, along with their matching certificates
* Import and use keys stored on smart cards
  * Yubikeys
  * [Virtual Smart Cards](https://learn.microsoft.com/en-us/windows/security/identity-protection/virtual-smart-cards/virtual-smart-card-overview)
  * ...any other smart card or PIV applet supported by the `Microsoft Smart Card Key Storage Provider`
* A nice-ish GUI for managing your hardware-backed keys
* Supports multiple SSH Agent listeners:
  * OpenSSH for Windows
  * PuTTY/Pageant
  * WSL2
  * Cygwin/mSys/MinGW
* Notifications so you know when your key is being used
* Configurable PIN/Password cache, so you don't have to re-enter your PIN/Password for rapid successive key usage (not available for WebAuthN keys)
* Support for [OpenSSH Certificates](https://smallstep.com/blog/use-ssh-certificates/)
  * Adds support for OpenSSH certificates to PuTTY!

## Getting Started

* Download and launch the latest [release](https://github.com/unreality/nCryptAgent/releases)
* Click **Create Key** and enter a key name and container name.
  * **Key Name** is a friendly descriptive name for the SSH Key
  * **Container Name** is the nCrypt key container identifier which will be used - it will be shown in the password prompt when signing is requested.
* Select your **Key Algorithm**
* Enter a **Password or PIN**
  * This can be empty if you wish to be a bit less secure
* Click **Save**
* You now have a new SSH key, you can click the **Copy Key** button to copy the `authorized_keys` content to the clipboard and save it to the remote server. Alternatively you can copy the public key's path for use as a command line arg, or opening with another program.

You can use the key by configuring your SSH client to use nCryptAgent as its SSH agent. For OpenSSH for Windows and PuTTY this should work automatically, as long as those listeners are enabled in the **Config** tab. For WSL2 and Cygwin, you will need to set your `SSH_AUTH_SOCK` environment variable. The commands for doing this are available in the **Config** tab.

## Getting Started with WebAuthN Security Keys

* From the nCryptAgent main window, select the dropdown arrow in the bottom left and click on **Create new WebAuthN key**
* Enter a friendly name for the key and choose a **Key Algorithm**
* Click **Save** and you will be prompted to enter your pin and touch your security key
* Your key is now available for use

`sk-ssh-ed25519@openssh.com` and `sk-ecdsa-sha2-nistp256@openssh.com` keys require [OpenSSH 8.4](https://www.openssh.com/txt/release-8.4) or higher to use.

### WebAuthN/FIDO Specific options

OpenSSH has a few specific options for `sk-ssh-ed25519@openssh.com` and `sk-ecdsa-sha2-nistp256@openssh.com` key types. nCryptAgent supports `verify-required`, but unfortunately Windows always demands a touch event if possible, so `no-touch-required` has no effect. To use `verify-required`, when creating your WebAuthN key select the **User Verification Required** option. The appropriate options flag will be added to the key when you click the **Copy Key** button ready for pasting into your `authorized_keys` file.

### Resident Keys

You can create a resident key by selecting the appropriate checkbox when creating the key. Unfortunately the Windows WebAuthN API doesn't support retrieving the required Public Key information from security keys.

## Getting Started with Smart Cards

If you already have a certificate and key on your smart card, you can skip to _Import an existing key_, otherwise you will need to create a certificate and key:

### Yubikeys

* Open the [YubiKey Manager](https://www.yubico.com/support/download/yubikey-manager/) App
* Select the PIV application
* Click on **Configure Certificates**
* Select an empty slot and click **Generate**
* Click through the wizard to create a self-signed certificate
* Once you have a certificate saved, follow the _Import an existing key_ section below.

### TPM Smart Cards

* Create a card if you don't have one:
  * Ensure your TPM is enabled in BIOS or UEFI. Different manufacturers name the setting differently.
  * Open a command prompt
  * Run `tpmvscmgr create /name <Friendly_Name> /AdminKey DEFAULT /pin PROMPT /pinpolicy minlen 4 /generate` where `<Friendly_Name>` is a name you choose
* You can use `certreq` and `certutil` to load a certificate onto the smart card, after which you can **Add existing nCrypt Key** to import your Smart Card credentials into nCryptAgent

## Import an existing key

If you have a key on your smart card (for instance you have existing credentials on your Yubikey), or have previously created a key using PCP, you can import that key by clicking on the dropdown next to **Create Key** and selecting **Add existing nCrypt key**. Select your key from the dropdown after selecting the provider and smart card reader (if required), and enter a name. Click **Save** and your existing key will be ready for use.

## Client Configuration

Once you have a key added to nCryptAgent you can use it by configuring your SSH client to use nCryptAgent as its SSH agent. For OpenSSH for Windows and PuTTY this should work automatically, as long as those listeners are enabled in the **Config** tab. For WSL2 and Cygwin, you will need to set your `SSH_AUTH_SOCK` environment variable. The commands for doing this are available in the **Config** tab.

* If you are using the **Named Pipe** listener, ensure the `OpenSSH Authentication Agent` service is stopped in `Services`
* If you are using the **Pageant** listener, ensure pageant is not running

## OpenSSH Certificates

Since `ssh-add` does [not support adding certificates without a private key](https://bugzilla.mindrot.org/show_bug.cgi?id=3212), nCryptAgent checks for a matching certificate in its `PublicKeys` directory (`%AppData%\nCryptAgent\PublicKeys`). If you have an OpenSSH certificate you wish to use, you can either use the `Add Cert` button to attach a certificate to the currently selected key, or alternatively place the certificate in the `PublicKeys` directory with the correct name. The name format for certificates is `<MatchingCertificateFingerprint>-cert.pub`. 

For example, if an nCrypt key has a location of `%AppData%\nCryptAgent\PublicKeys\deadbeefd530ca2d01b3b74c8641fe29.pub` the matching certificate will be named `%AppData%\nCryptAgent\PublicKeys\deadbeefd530ca2d01b3b74c8641fe29-cert.pub`. 

## Building

* To build you'll need `windres` which can be obtained by downloading the latest release of [llvm-mingw](https://github.com/mstorsjo/llvm-mingw)
* Download go deps with `go mod tidy`
* `windres.exe -i resources.rc -o rsrc.syso -O coff`
* `go build -ldflags "-H=windowsgui" -o build\nCryptAgent.exe`

I'll get around to making a proper build script at some point...

## Known Issues

* Sometimes the PIN prompt does not obtain focus correctly and will pop up in the background.
* Sometimes the 'Copy' buttons do not correctly copy to clipboard.

## FAQ

### I'd REALLY like to use a non-hardware key
If you simply MUST have a software key you can open the configuration file at `%AppData%\nCryptAgent\config.json` and add a key with `providerName: "Microsoft Software Key Storage Provider"`  and set the `containerName` to an existing key. You can get a list of existing keys by running `certutil -key -user -csp KSP` in a command prompt window.

### The nCrypt `containerName` lists a location on my local filesystem, what gives?
The `Platform Crypto Provider` does not actually store the complete key in the TPM, instead it stores a file for loading into the TPM when signing operations are required. The files are specific to each TPM so your key is still non-exportable. [@ElMostafaIdrassi](https://github.com/ElMostafaIdrassi/pcpcrypto#general-trivia-about-pcp-tpm-keys) has written a nice explanation of it if you'd like more detail.

