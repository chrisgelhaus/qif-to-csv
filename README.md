# qif-to-csv
Quicken QIF File Extraction Utility

# Description
This is an initial release of a tool to take a Quicken export file in QIF format and extract the transactions into a common CSV file. The current CSV format is compatible with Monarch Money; However it needs to be updated to allow the export fields to be selected at runtime to allow compatibility with any solution. 

# Examples
qif-to-csv.exe extract -categories -payees -accounts -tags -inputFile "filename"

qif-to-csv.exe convert -inputFile "FileName" -accountName "Account" -outputFile "Filename"

