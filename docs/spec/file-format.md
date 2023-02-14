# File Format Specification of Diag Packaged Data Set
> The `.diag` file format

## Preamble
Some metadata needs to be uploaded to the [Clinic server](https://clinic.pingcap.com/) before the actual data set is uploaded. Typically the user might not have enough permission to upload a data set to a certain cluster, checking before actually transferring the file could save time and bandwidth.

Considering the nature that a packaged file is not necessarily created on the same system as it is uploaded to the Clinic server, we are not storing the metadata in a separate file (which would make the upload procedure more complex for the user and the small metadata file could easily be lost).

This documentation tries to define a file format that combines metadata needed and the packaged data set together in one file.

## Header
### File Signature
The file starts with a magic number as file signature, to identify it is a `.diag` format file. The magic number is a `4-bytes` constant:
```
44 31 61 67 // b’D1ag’
```

### Package Format Type (1 byte)
One byte is used to indicate the type and format version of the file. The byte is an `8-bits` ***binary map*** that uses the lower 6 bits as two groups of indicators and the highest 2 bits reserved for future usage.

Format of a package indicates the file format specification version, so that the diag client knows how to parse its data.

Compression indicates which algorithm is used to compress the payload.

The layout is as follows:

| Bit No. |       1-2       |    3-5    |     6-8     |
| :----:  |      :----:     |   :----:  |    :----:   |
|  Usage  | Package Version | Data Type | Compression |


If either data type index or compression type index exceeds their `3-bits` space, the 2 bits of the package version may be used to indicate a different layout of the lower 6 bits, or to indicate a larger type bits space, as defined by the package versions.

Package Version:
 - `00`: always be `00` until there is any incompatible change of the package format byte(s) layout in the future

Data type bits may take the following values:
 - `000`: unknown or undefined format, should never be seen
 - `001`: the legacy format used before Diag `v0.7.0`, should never be seen as it don’t have this header
 - `010`: the diag format version 2, the format defined in this documentation, payload not encrypted
 - `011`: the diag format version 2, the format defined in this documentation, payload is encrypted with key of Clinic server

Compression bits may take the following values:
 - `000`: none, the payload is not compressed, equivalent to `.tar`
 - `001`: the payload is compressed with ***gzip***, equivalent to `.tar.gz`
 - `010`: the payload is compressed with ***Zstandard***, equivalent to `.tar.zst`

For example, a package with the latest file format, data set encrypted and compressed with Zstandard would have the type bits: `00 011 010`, or `26` in decimal, or `1a` in hexadecimal.

### The Payload Offset (3 bytes)
Following the file signature and type is a `3-bytes`(`24-bits`) field storing an ***unsigned integer*** that represents the length of the encrypted metadata.

So that the payload starts from the ***byte*** number `(4 + 1 + 3 + offset value)`.

The maximum length of encrypted metadata is ***16MB***.

## Metadata
Metadata could be any data that the Clinic server knows how to parse, with length less than ***16MB*** after encryption.

However, to simplify things, we define it as a plain ***JSON string*** encrypted with the same format as payload, which is an AES key encrypted by the RSA public key of the Clinic server, following the plain ***JSON string*** encrypted by the AES key.

Note that the AES key used to encrypt the metadata is different from the one used to encrypt payload. But they are encrypted by the same RSA public key from the Clinic server.

> If the ***16MB*** of plain ***JSON string*** (and encrypted RSA-OAEP AES key block) is not enough to hold all the metadata data we need (which is highly not possible), we could change this to a compressed JSON data.

## Payload
### The Encryption Key
The encryption key is the RSA encrypted AES key which is used to encrypt the data set.

As the AES key used is `32 bytes` long (`AES-256`), and a typical RSA key is `2048 bytes` long, the encrypted key is the same length as the RSA key, not a multiple of it.

### Packaged Data Set
The packed data set is the actual payload of the file, it is an AES encrypted archive file, whis is archived with ***tar*** and then compressed with ***Zstandard*** (`.tar.zst`) by default.

## Compatibility
### Legacy File Format
The legacy format of the `.diag` file we used before `v0.7.x` does not have metadata bundled, but it is in a similar structure as the one described in this documentation.

| Length | 4-bytes | 1-byte | 3-bytes | RSA Key Length | Varies | RSA Key Length |  Varies  |
| :----: | :----:  | :----: | :----:  |     :----:     | :----: |     :----:     |  :----:  |
| Legacy |  None   |  None  |  None   |      None      |  None  |    AES Key     | Data Set |
| Current | `44 31 61 67` | Type | Metadata Length | Metadata AES Key | Metadata JSON | Payload AES Key | Data Set |

The encryption key of Clinic server must be acquired and configured before the data set is able to be packaged to a `.diag` file.
 