# gpugen-dist

## Pre-requisites
Working OpenCL and/or CUDA installation (if you can run hashcat, you can run this)

## Releases
Pre-built binaries are in [releases](https://github.com/blurbdust/gpugen-dist/releases). These are the wrappers as talked about on [Twitter](https://twitter.com/Blurbdust/status/1660827973929299968).

The clients for Linux and Windows when run, will automatically download the software, check to ensure generation works, reach out to `genrt.blurbdust.pw` to check out a table index, generate the table, and upload the table. If you would prefer not to upload tables directly, that's completely fine and you'll need to modify the client. 
If the upload fails for whatever reason, reach out to me on Twitter or Discord and we can figure out how to send the ~2GB file. 

Once all tables have been generated, I will batch convert from `.rt` to `.rtc` to save on storage space and distribute tables. I'm expecting <= 4TB for final storage.

Please don't DoS the index server, it's fragile.

## How To
`.\client.winx64.exe` or `./client_linx64` it should be that easy 

## Compatibility
The tables should be compatible with `rcrack` and the recreation [here](https://github.com/inAudible-NG/RainbowCrack-NG). I do already have plugins written and will release those closer to completion of generation.

## Credits
Everyone who has published code or spoke about this concept at previous conferences. If you want your individual name or handle listed here, I'm happy to accommodate. 
