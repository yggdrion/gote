Add-Type -AssemblyName System.Drawing

# Load the PNG image
$png = [System.Drawing.Image]::FromFile("c:\Users\rapha\workspace\gote\build\appicon.png")

# Create a bitmap from the image with the desired size (32x32 for ICO)
$bitmap = New-Object System.Drawing.Bitmap($png, 32, 32)

# Get the icon handle
$handle = $bitmap.GetHicon()

# Create an icon from the handle
$icon = [System.Drawing.Icon]::FromHandle($handle)

# Save as ICO file
$iconPath = "c:\Users\rapha\workspace\gote\build\windows\icon.ico"
$fileStream = [System.IO.FileStream]::new($iconPath, [System.IO.FileMode]::Create)
$icon.Save($fileStream)

# Clean up
$fileStream.Close()
$bitmap.Dispose()
$png.Dispose()
$icon.Dispose()

Write-Host "Icon converted successfully to $iconPath"
