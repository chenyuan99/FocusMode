# Winget Submission

FocusMode includes draft winget manifests under:

```text
winget/manifests/c/ChenYuan/FocusMode/0.1.0/
```

The package identifier is:

```text
ChenYuan.FocusMode
```

After the manifests are submitted and accepted into `microsoft/winget-pkgs`, users can install FocusMode with:

```powershell
winget install ChenYuan.FocusMode
```

## Submission Steps

1. Fork or clone `https://github.com/microsoft/winget-pkgs`.
2. Copy this repo's `winget/manifests/c/ChenYuan/FocusMode/0.1.0/` folder into the same path in the `winget-pkgs` checkout.
3. Validate the manifest with winget tooling:

```powershell
winget validate .\manifests\c\ChenYuan\FocusMode\0.1.0
```

4. Commit the manifest folder.
5. Open a pull request to `microsoft/winget-pkgs`.

## Release Asset

The v0.1.0 manifest points to:

```text
https://github.com/chenyuan99/FocusMode/releases/download/v0.1.0/focusmode-windows-amd64.zip
```

SHA256:

```text
10A8BB4879C7567F6411510E5E05BEFA002A978BD4C97BEFF3DB615632ADF499
```

