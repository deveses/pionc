param (
    [string]$InputFile,  # The input file to process
    [string]$OutputFile  # The output file after processing
)

if (-not (Test-Path $InputFile)) {
    Write-Error "Input file '$InputFile' does not exist."
    exit 1
}

$insideBlock = $false

Get-Content $InputFile | ForEach-Object {
    if ($_ -match '#ifdef _MSC_VER') { 
        $insideBlock = $true          
    }
    if (-not $insideBlock) { 
        $_
    }
    else {
        '//'+$_
    }
    if ($insideBlock -and ($_ -match '#endif')) { 
        $insideBlock = $false         
    }
} | Set-Content $OutputFile

Write-Host "Processed '$InputFile' and saved to '$OutputFile'."
