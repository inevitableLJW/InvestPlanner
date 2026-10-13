param(
    [string]$SourceDirectory = (Join-Path $PSScriptRoot 'source'),
    [string]$OutputDirectory = (Join-Path $PSScriptRoot '../../web/public/target-icons')
)

$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Drawing
New-Item -ItemType Directory -Path $OutputDirectory -Force | Out-Null

foreach ($sourceFile in Get-ChildItem -LiteralPath $SourceDirectory -Filter '*.png' -File) {
    $source = [System.Drawing.Bitmap]::new($sourceFile.FullName)
    $output = [System.Drawing.Bitmap]::new(128, 128, [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
    $graphics = [System.Drawing.Graphics]::FromImage($output)
    $attributes = [System.Drawing.Imaging.ImageAttributes]::new()
    try {
        $graphics.Clear([System.Drawing.Color]::Transparent)
        $graphics.CompositingMode = [System.Drawing.Drawing2D.CompositingMode]::SourceCopy
        $graphics.CompositingQuality = [System.Drawing.Drawing2D.CompositingQuality]::HighQuality
        $graphics.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
        $graphics.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
        $attributes.SetWrapMode([System.Drawing.Drawing2D.WrapMode]::TileFlipXY)
        $scale = [Math]::Min(128 / $source.Width, 128 / $source.Height)
        $width = [int][Math]::Round($source.Width * $scale)
        $height = [int][Math]::Round($source.Height * $scale)
        $destination = [System.Drawing.Rectangle]::new([int]((128 - $width) / 2), [int]((128 - $height) / 2), $width, $height)
        $graphics.DrawImage($source, $destination, 0, 0, $source.Width, $source.Height, [System.Drawing.GraphicsUnit]::Pixel, $attributes)
        $outputPath = Join-Path $OutputDirectory $sourceFile.Name
        $output.Save($outputPath, [System.Drawing.Imaging.ImageFormat]::Png)
        [pscustomobject]@{ Name = $sourceFile.Name; Width = $output.Width; Height = $output.Height; CornerAlpha = $output.GetPixel(0, 0).A; Bytes = (Get-Item -LiteralPath $outputPath).Length }
    }
    finally {
        $attributes.Dispose()
        $graphics.Dispose()
        $output.Dispose()
        $source.Dispose()
    }
}
