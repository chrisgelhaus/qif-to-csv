package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

func main() {
	// Flag Variables
	inputFileName := ""
	outputFileName := ""
	categoryMappingFile := ""
	payeeMappingFile := ""
	accountMappingFile := ""
	extractCategoryFlag := false
	extractPayeeFlag := false
	extractTagFlag := false
	extractAccountFlag := false

	_ = outputFileName

	// Flagsets
	extractCmd := flag.NewFlagSet("extract", flag.ExitOnError)
	extractCategory := extractCmd.Bool("categories", false, "categories")
	extractPayee := extractCmd.Bool("payees", false, "payees")
	extractTag := extractCmd.Bool("tags", false, "tags")
	extractAccount := extractCmd.Bool("accounts", false, "accounts")
	extractInputFile := extractCmd.String("inputfile", "", "inputfile")

	convertCmd := flag.NewFlagSet("convert", flag.ExitOnError)
	convertInputFile := convertCmd.String("inputfile", "", "inputfile")
	convertAccountName := convertCmd.String("accountname", "", "accountname")
	convertOutputFile := convertCmd.String("outputfile", "", "outputfile")
	convertCategoryMapFile := convertCmd.String("categorymap", "", "categorymap")
	convertPayeeMapFile := convertCmd.String("payeemap", "", "payeemap")
	convertAccountMapFile := convertCmd.String("accountmap", "", "accountmap")

	if len(os.Args) < 2 {
		fmt.Println("expected 'extract' or 'convert' subcommands")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "extract":
		extractCmd.Parse(os.Args[2:])
		fmt.Println("subcommand 'extract'")
		fmt.Println("	Extract Categories:", *extractCategory)
		fmt.Println("	Extract Payees:", *extractPayee)
		fmt.Println("	Extract Tags:", *extractTag)
		fmt.Println("	Extract Accounts:", *extractAccount)
		fmt.Println("	Source File:", *extractInputFile)
		fmt.Println("	Args:", extractCmd.Args())
		extractCategoryFlag = *extractCategory
		extractPayeeFlag = *extractPayee
		extractTagFlag = *extractTag
		extractAccountFlag = *extractAccount
		inputFileName = *extractInputFile
	case "convert":
		convertCmd.Parse(os.Args[2:])
		fmt.Println("subcommand 'convert'")
		fmt.Println("	inputfile:", *convertInputFile)
		fmt.Println("	accountname:", *convertAccountName)
		fmt.Println("	outputfile:", *convertOutputFile)
		fmt.Println("	applycategorymap:", *convertCategoryMapFile)
		fmt.Println("	applypayeemap:", *convertPayeeMapFile)
		fmt.Println("	applyaccountmap:", *convertAccountMapFile)
		//fmt.Println("	tail:", convertCmd.Args())
		//accountName = *convertAccountName
		inputFileName = *convertInputFile
		outputFileName = *convertOutputFile
		categoryMappingFile = *convertCategoryMapFile
		payeeMappingFile = *convertPayeeMapFile
		accountMappingFile = *convertAccountMapFile
	default:
		fmt.Println("expected 'extract' or 'convert' subcommands")
		os.Exit(1)
	}

	if os.Args[1] == "extract" {
		if extractCategoryFlag {
			err := extractCategories(inputFileName, "categoryList.txt")
			if err != nil {
				fmt.Println("Error with category extraction: ", err)
			}
		}
		if extractPayeeFlag {
			err := extractPayees(inputFileName, "payeeList.txt")
			if err != nil {
				fmt.Println("Error with payee extraction: ", err)
			}
		}
		if extractTagFlag {
			err := extractTags(inputFileName, "tagsList.txt")
			if err != nil {
				fmt.Println("Error with tag extraction: ", err)
			}
		}
		if extractAccountFlag {
			err := extractAccounts(inputFileName, "AccountsList.txt")
			if err != nil {
				fmt.Println("Error with account extraction: ", err)
			}
		}
	}

	if os.Args[1] == "convert" {
		exportTransactions(inputFileName, outputFileName, categoryMappingFile, payeeMappingFile, accountMappingFile)
	}
}

func exportTransactions(inputFileName string, outputFileName string, categoryMappingFile string, payeeMappingFile string, accountMappingFile string) {
	// Output CSV Header
	var transactionRegexString string = `D(?<month>\d{1,2})\/(\s?(?<day>\d{1,2}))'(?<year>\d{2})[\r\n]+(U(?<amount1>.*?)[\r\n]+)(T(?<amount2>.*?)[\r\n]+)(C(?<cleared>.*?)[\r\n]+)((N(?<number>.*?)[\r\n]+)?)(P(?<payee>.*?)[\r\n]+)((M(?<memo>.*?)[\r\n]+)?)(L(?<category>.*?)[\r\n]+)`
	var accountBlockHeaderRegex string = `(?m)^!Account[^\n]*\n^N(.*?)\n^T(.*?)\n^\^\n^!Type:(Bank|CCard)\s*\n`
	outputCSVHeader := "Date,Merchant,Category,Account,Original Statement,Notes,Amount,Tags\n"
	var categoryMapping map[string]string
	var payeeMapping map[string]string
	var accountMapping map[string]string
	var err error

	//// Create the output file.
	//outputFile, err := os.Create(outputFileName)
	//if err != nil {
	//	fmt.Println("Error creating file:", err)
	//	return
	//}
	//defer outputFile.Close()

	// Write header to the output file.
	//_, err = outputFile.WriteString(outputCSVHeader)
	//if err != nil {
	//	fmt.Println("Error writing to file:", err)
	//	return
	//}

	// Load the Category Mapping
	if categoryMappingFile != "" {
		categoryMapping, err = loadMapping(categoryMappingFile)
		if err != nil {
			fmt.Println("Error loading mapping:", err)
			return
		}
		fmt.Println("Mapping loaded:")
		for k, v := range categoryMapping {
			fmt.Printf("  %s -> %s\n", k, v)
		}
	} else {
		fmt.Println("No category mapping file loaded:", err)
	}

	// Load the Payee Mapping
	if payeeMappingFile != "" {
		payeeMapping, err := loadMapping(payeeMappingFile)
		if err != nil {
			fmt.Println("Error loading mapping:", err)
			return
		}
		fmt.Println("Mapping loaded:")
		for k, v := range payeeMapping {
			fmt.Printf("  %s -> %s\n", k, v)
		}
	} else {
		fmt.Println("No payee mapping file loaded:", err)
	}

	// Load the Account Mapping
	if accountMappingFile != "" {
		accountMapping, err = loadMapping(accountMappingFile)
		if err != nil {
			fmt.Println("Error loading mapping:", err)
			return
		}
		fmt.Println("Account Mapping loaded:")
		for k, v := range accountMapping {
			if v != "" {
				fmt.Printf("  %s -> %s\n", k, v)
			}
		}
	} else {
		fmt.Println("No account mapping file loaded:", err)
	}

	// Open the input file and find all the Bank and CCard blocks
	// Load input file
	inputBytes, err := os.ReadFile(inputFileName)
	if err != nil {
		fmt.Println("Error reading file:", err)
	} else {
		fmt.Printf("Input file opened. Length: %d\n", len(inputBytes))
	}
	inputContent := string(inputBytes)

	// Standardize Line Endings to simplify Regex
	inputContent = strings.ReplaceAll(inputContent, "\r\n", "\n")

	// Gather the Account Blocks
	// Compile the regex
	regex, err := regexp.Compile(accountBlockHeaderRegex)
	if err != nil {
		return
	}
	accountBlocks := regex.FindAllStringSubmatchIndex(inputContent, -1)
	if len(accountBlocks) == 0 {
		fmt.Println("No matches found.")
	}

	// loop over each account block
	// Find all matches for transactions
	for _, accountBlock := range accountBlocks {
		var outputAccountName string
		accountName := inputContent[accountBlock[2]:accountBlock[3]]
		if len(accountMapping[accountName]) > 0 {
			outputAccountName = accountMapping[accountName]
		} else {
			outputAccountName = accountName
		}

		restOfText := inputContent[accountBlock[1]:]
		nextTypePattern := `(?mi)^\s*!Type:.*$`
		nextTypeRe := regexp.MustCompile(nextTypePattern)
		nextLoc := nextTypeRe.FindStringIndex(restOfText)
		var endPos int
		if nextLoc != nil {
			endPos = accountBlock[1] + nextLoc[0]
		} else {
			endPos = len(inputContent)
		}
		//fmt.Printf("%d, %d\n", accountBlock[1], nextLoc[0])

		// Create unique output file per Account
		outputFile, err := os.Create(accountName + outputFileName)
		if err != nil {
			fmt.Println("Error creating file:", err)
			return
		}
		// Write header to the output file.
		_, err = outputFile.WriteString(outputCSVHeader)
		if err != nil {
			fmt.Println("Error writing to file:", err)
			return
		}
		//defer outputFile.Close()

		// Extract the text between the type lines
		textBetweenTypes := inputContent[accountBlock[1]:endPos]

		// Use the existing pattern to match entries
		regex, err := regexp.Compile(transactionRegexString)
		if err != nil {
			return
		}

		// Find all transactions in the content.
		transactions := regex.FindAllStringSubmatch(textBetweenTypes, -1)

		for _, t := range transactions {
			// Check if there is a captured group and extract the content.
			if len(t) > 1 { // Ensure there is a captured group.
				//fmt.Println("Transaction:")

				month := strings.TrimSpace(t[1])
				day := strings.TrimSpace(t[2])
				year := strings.TrimSpace(t[4])
				amount1 := strings.TrimSpace(t[6])
				amount2 := strings.TrimSpace(t[8])
				cleared := strings.TrimSpace(t[10])
				number := strings.TrimSpace(t[13])
				payee := strings.TrimSpace(t[15])
				transactionMemo := strings.TrimSpace(t[18])
				category, tag := splitCategoryAndTag(t[20])

				if len(payeeMapping) > 0 {
					payee = applyMapping(payee, payeeMapping)
				}
				if len(categoryMapping) > 0 {
					category = applyMapping(category, categoryMapping)
				}

				// DATE
				fullYear := "20" + year
				month = "0" + month
				fullMonth := month[len(month)-2:]
				day = "0" + day
				fullDay := day[len(day)-2:]
				fullDate := fullYear + "-" + fullMonth + "-" + fullDay

				amount1 = prepareString(amount1)
				amount2 = prepareString(amount2)
				cleared = prepareString(cleared)
				number = prepareString(number)
				payee = prepareString(payee)
				transactionMemo = prepareString(transactionMemo)
				category = prepareString(category)
				tag = prepareString(tag)

				//fmt.Printf("  Date: %s-%s-%s\n", fullMonth, fullDay, fullYear)
				//fmt.Printf("  Payee: %s\n", payee)
				//fmt.Printf("  Category: %s\n", category)
				//fmt.Printf("  AccountName: %s\n", outputAccountName)
				//fmt.Printf("  Amount: %s\n", amount1)
				//fmt.Printf("  Amount2: %s\n", amount2)
				//if transactionMemo != "" {
				//	fmt.Printf("  Transaction Memo: %s\n", transactionMemo)
				//}
				//if number != "" {
				//	fmt.Printf("  Transaction Number: %s\n", number)
				//}
				//if cleared != "" {
				//	fmt.Printf("  Cleared: %s\n", cleared)
				//}
				//if tag != "" {
				//	fmt.Printf("  Tag: %s\n", tag)
				//}

				_, err := outputFile.WriteString(fullDate + "," + payee + "," + category + "," + outputAccountName + "," + payee + "," + transactionMemo + "," + amount1 + "," + tag + "\n")

				if err != nil {
					fmt.Println("Error writing to file:", err)
					return
				}
			}
		}
		outputFile.Close()
	}

}

func extractPayees(inputFileName string, outputFileName string) error {
	// Changing up the process
	// Each task will have all processes within it to make parameter adjustments easier
	// 1. Get input file
	// 1. Gather payees from the !Type:Bank|CCard regiser entries
	// 2. populate the output file

	var payees []string
	var transactionRegexString string = `D(?<month>\d{1,2})\/(\s?(?<day>\d{1,2}))'(?<year>\d{2})[\r\n]+(U(?<amount1>.*?)[\r\n]+)(T(?<amount2>.*?)[\r\n]+)(C(?<cleared>.*?)[\r\n]+)((N(?<number>.*?)[\r\n]+)?)(P(?<payee>.*?)[\r\n]+)((M(?<memo>.*?)[\r\n]+)?)(L(?<category>.*?)[\r\n]+)`
	var accountBlockHeaderRegex string = `(?m)^!Account[^\n]*\n^N(.*?)\n^T(.*?)\n^\^\n^!Type:(Bank|CCard)\s*\n`

	// Create the category output file
	payeeFile, err := os.Create(outputFileName)
	if err != nil {
		fmt.Println("Error creating category file:", err)
		return err
	} else {
		fmt.Println("Created catergory output file.")
	}
	defer payeeFile.Close()

	// Load input file
	inputBytes, err := os.ReadFile(inputFileName)
	if err != nil {
		fmt.Println("Error reading file:", err)
	} else {
		fmt.Printf("Input file opened. Length: %d\n", len(inputBytes))
	}
	inputContent := string(inputBytes)

	// Standardize Line Endings to simplify Regex
	inputContent = strings.ReplaceAll(inputContent, "\r\n", "\n")

	// Gather payees from the Accounts
	// Compile the regex
	regex, err := regexp.Compile(accountBlockHeaderRegex)
	if err != nil {
		return err
	}
	accountBlocks := regex.FindAllStringSubmatchIndex(inputContent, -1)
	if len(accountBlocks) == 0 {
		fmt.Println("No matches found.")
	}

	// loop over each account block and pull out payees
	for _, accountBlock := range accountBlocks {
		accountName := inputContent[accountBlock[2]:accountBlock[3]]

		restOfText := inputContent[accountBlock[1]:]
		nextTypePattern := `(?mi)^\s*!Type:.*$`
		nextTypeRe := regexp.MustCompile(nextTypePattern)
		nextLoc := nextTypeRe.FindStringIndex(restOfText)
		var endPos int
		if nextLoc != nil {
			endPos = accountBlock[1] + nextLoc[0]
		} else {
			endPos = len(inputContent)
		}

		// Extract the text between the type lines
		textBetweenTypes := inputContent[accountBlock[1]:endPos]

		// Use the existing pattern to match entries
		regex, err := regexp.Compile(transactionRegexString)
		if err != nil {
			return nil
		}

		// Find all matches in the content.
		transactions := regex.FindAllStringSubmatch(textBetweenTypes, -1)
		fmt.Printf("%d payees extracted from account: %s\n", len(transactions), accountName)

		// Loop through matches and add payees to the array
		for _, t := range transactions {
			if len(t) > 1 {
				payee := strings.TrimSpace(t[15])
				payee = prepareString(payee)
				payees = append(payees, payee)
			}
		}
	}

	// Sort and dedupe payee list
	outputPayeeList := sortAndDedupStrings(payees)
	// Write payees to the file
	for _, item := range outputPayeeList {
		_, err := payeeFile.WriteString(item + "\n")
		if err != nil {
			fmt.Printf("Error Writing to category file:\n")
		}
	}

	fmt.Println("Extracted Payees: ", len(outputPayeeList))

	return nil
}

func extractAccounts(inputFileName string, outputFileName string) error {
	// Changing up the process
	// Each task will have all processes within it to make parameter adjustments easier
	// 1. Get input file
	// 1. Gather Account !Type:Bank|CCard regiser entries
	// 2. populate the output file

	var accountNames []string
	var accountBlockHeaderRegex string = `(?m)^!Account[^\n]*\n^N(.*?)\n^T(.*?)\n^\^\n^!Type:(Bank|CCard)\s*\n`

	// Create the account output file
	accountFile, err := os.Create(outputFileName)
	if err != nil {
		fmt.Println("Error creating account file:", err)
		return err
	} else {
		fmt.Println("Created account output file.")
	}
	defer accountFile.Close()

	// Load input file
	inputBytes, err := os.ReadFile(inputFileName)
	if err != nil {
		fmt.Println("Error reading file:", err)
	} else {
		fmt.Printf("Input file opened. Length: %d\n", len(inputBytes))
	}
	inputContent := string(inputBytes)

	// Standardize Line Endings to simplify Regex
	inputContent = strings.ReplaceAll(inputContent, "\r\n", "\n")

	// Gather the Accounts
	// Compile the regex
	regex, err := regexp.Compile(accountBlockHeaderRegex)
	if err != nil {
		return err
	}
	accountBlocks := regex.FindAllStringSubmatchIndex(inputContent, -1)
	if len(accountBlocks) == 0 {
		fmt.Println("No matches found.")
	}

	// loop over each account block and pull out payees
	for _, accountBlock := range accountBlocks {
		accountName := inputContent[accountBlock[2]:accountBlock[3]]
		accountName = strings.TrimSpace(accountName)
		accountName = prepareString(accountName)
		accountNames = append(accountNames, accountName)
	}

	// Sort and dedupe payee list
	outputAccountList := sortAndDedupStrings(accountNames)
	// Write payees to the file
	for _, item := range outputAccountList {
		_, err := accountFile.WriteString(item + "\n")
		if err != nil {
			fmt.Printf("Error Writing to account file:\n")
		}
	}

	fmt.Println("Extracted Account: ", len(outputAccountList))

	return nil
}

func extractCategories(inputFileName string, outputFileName string) error {
	// Changing up the process
	// Each task will have all processes within it to make parameter adjustments easier
	// 1. Get input file
	// 1. Gather caterories from the !Type:Cat block
	// 1. Gather categories from the !Type:Bank|CCard regiser entries
	// 2. populate the output file

	var categories []string
	var transactionRegexString string = `D(?<month>\d{1,2})\/(\s?(?<day>\d{1,2}))'(?<year>\d{2})[\r\n]+(U(?<amount1>.*?)[\r\n]+)(T(?<amount2>.*?)[\r\n]+)(C(?<cleared>.*?)[\r\n]+)((N(?<number>.*?)[\r\n]+)?)(P(?<payee>.*?)[\r\n]+)((M(?<memo>.*?)[\r\n]+)?)(L(?<category>.*?)[\r\n]+)`
	var catRecordRegex string = `(?m)(^N(.*)\n^D(.*)\n(^T(.*)\n)?(^[R,E](.*)\n)?(^I(.*)\n)?^\^\n)`
	var catBlockHeaderRegex string = `(?m)^!Type:Cat\n`
	var accountBlockHeaderRegex string = `(?m)^!Account[^\n]*\n^N(.*?)\n^T(.*?)\n^\^\n^!Type:(Bank|CCard)\s*\n`

	// Create the category output file
	categoryFile, err := os.Create(outputFileName)
	if err != nil {
		fmt.Println("Error creating category file:", err)
		return err
	} else {
		fmt.Println("Created catergory output file.")
	}
	defer categoryFile.Close()

	// Load input file
	inputBytes, err := os.ReadFile(inputFileName)
	if err != nil {
		fmt.Println("Error reading file:", err)
	} else {
		fmt.Printf("Input file opened. Length: %d\n", len(inputBytes))
	}
	inputContent := string(inputBytes)

	// Standardize Line Endings to simplify Regex
	inputContent = strings.ReplaceAll(inputContent, "\r\n", "\n")
	//fmt.Printf("Sample inputContent:\n%s\n", inputContent[:1000])

	// Find the position of the Category Block
	catTypeRe, err := regexp.Compile(catBlockHeaderRegex)
	if err != nil {
		fmt.Println("Error compiling regular expression: ", err)
	}
	loc := catTypeRe.FindStringIndex(inputContent)
	if loc == nil {
		fmt.Printf("No Category block found.\n")
		return nil
	} else {
		// Debugging output
		fmt.Printf("Category block found at position: %d\n", loc[1])
	}

	// Find the position of the next Type block
	restOfText := inputContent[loc[1]:]
	nextTypePattern := `(?mi)^\s*!Type:.*$`
	nextTypeRe := regexp.MustCompile(nextTypePattern)
	nextLoc := nextTypeRe.FindStringIndex(restOfText)
	fmt.Printf("Next type found at:%d\n", nextLoc[1])
	var endPos int
	if nextLoc != nil {
		// Found another Type line.
		endPos = loc[1] + nextLoc[0]
	} else {
		// No other Type found
		endPos = len(inputContent)
	}

	// Extract the text between the Type lines
	textBetweenTypes := inputContent[loc[1]:endPos]

	// Use the existing pattern to match entries
	regex, err := regexp.Compile(catRecordRegex)
	if err != nil {
		return err
	}

	// Find all matches in the content.
	matches := regex.FindAllStringSubmatch(textBetweenTypes, -1)
	fmt.Printf("%d entries extracted from the category block.\n", len(matches))

	// Extract patterns to array from the Category block.
	for _, t := range matches {
		// Check if there is a captured group and extract the content.
		if len(t) > 1 {
			// Ensure there is a captured group.
			category := strings.TrimSpace(t[2])
			categories = append(categories, category)
		}
	}

	// Gather categories from the Accounts
	// Compile the regex
	regex, err = regexp.Compile(accountBlockHeaderRegex)
	if err != nil {
		return err
	}
	accountBlocks := regex.FindAllStringSubmatchIndex(inputContent, -1)
	if len(accountBlocks) == 0 {
		fmt.Println("No matches found.")
	}

	// loop over each account block and pull out categories
	for _, accountBlock := range accountBlocks {
		// Find the next Type Block
		accountName := inputContent[accountBlock[2]:accountBlock[3]]
		restOfText := inputContent[accountBlock[1]:]
		nextTypePattern := `(?mi)^\s*!Type:.*$`
		nextTypeRe := regexp.MustCompile(nextTypePattern)
		nextLoc := nextTypeRe.FindStringIndex(restOfText)
		var endPos int
		if nextLoc != nil {
			endPos = accountBlock[1] + nextLoc[0]
		} else {
			endPos = len(inputContent)
		}

		// Extract the text between the type lines
		textBetweenTypes := inputContent[accountBlock[1]:endPos]

		// Use the existing pattern to match entries
		regex, err := regexp.Compile(transactionRegexString)
		if err != nil {
			return nil
		}

		// Find all matches in the content.
		transactions := regex.FindAllStringSubmatch(textBetweenTypes, -1)
		fmt.Printf("%d categories extracted from account: %s\n", len(transactions), accountName)

		// Loop through matches and add categories to the array
		for _, t := range transactions {
			if len(t) > 1 {
				category, _ := splitCategoryAndTag(t[20])
				category = prepareString(category)
				categories = append(categories, category)
			}
		}
	}

	// Sort and dedupe category list
	outputCategoryList := sortAndDedupStrings(categories)
	// Write categories to the file
	for _, item := range outputCategoryList {
		_, err := categoryFile.WriteString(item + "\n")
		if err != nil {
			fmt.Printf("Error Writing to category file:\n")
		}
	}

	fmt.Println("Extracted Categories: ", len(outputCategoryList))

	return nil
}

func extractTags(inputFileName string, outputFileName string) error {
	// Changing up the process
	// Each task will have all processes within it to make parameter adjustments easier
	// 1. Get input file
	// 1. Gather tags from the !Type:Tag block
	// 1. Gather tag from the !Type:Bank|CCard regiser entries
	// 2. populate the output file

	var tags []string
	var transactionRegexString string = `D(?<month>\d{1,2})\/(\s?(?<day>\d{1,2}))'(?<year>\d{2})[\r\n]+(U(?<amount1>.*?)[\r\n]+)(T(?<amount2>.*?)[\r\n]+)(C(?<cleared>.*?)[\r\n]+)((N(?<number>.*?)[\r\n]+)?)(P(?<payee>.*?)[\r\n]+)((M(?<memo>.*?)[\r\n]+)?)(L(?<category>.*?)[\r\n]+)`
	var tagRecordRegex string = `(?m)(^N(.*)\n^(D(.*)\n^)?\^\n)`
	var tagBlockHeaderRegex string = `(?m)^!Type:Tag\n`
	var accountBlockHeaderRegex string = `(?m)^!Account[^\n]*\n^N(.*?)\n^T(.*?)\n^\^\n^!Type:(Bank|CCard)\s*\n`

	// Create the tag output file
	tagFile, err := os.Create(outputFileName)
	if err != nil {
		fmt.Println("Error creating tag file:", err)
		return err
	} else {
		fmt.Println("Created tag output file.")
	}
	defer tagFile.Close()

	// Load input file
	inputBytes, err := os.ReadFile(inputFileName)
	if err != nil {
		fmt.Println("Error reading file:", err)
	} else {
		fmt.Printf("Input file opened. Length: %d\n", len(inputBytes))
	}
	inputContent := string(inputBytes)

	// Standardize Line Endings to simplify Regex
	inputContent = strings.ReplaceAll(inputContent, "\r\n", "\n")

	// Find the position of the Tag Block
	tagTypeRe, err := regexp.Compile(tagBlockHeaderRegex)
	if err != nil {
		fmt.Println("Error compiling regular expression: ", err)
	}
	loc := tagTypeRe.FindStringIndex(inputContent)
	if loc == nil {
		fmt.Printf("No Tag block found.\n")
		return nil
	} else {
		// Debugging output
		fmt.Printf("Tag block found at position: %d\n", loc[1])
	}

	// Find the position of the next Type block
	restOfText := inputContent[loc[1]:]
	nextTypePattern := `(?mi)^\s*!Type:.*$`
	nextTypeRe := regexp.MustCompile(nextTypePattern)
	nextLoc := nextTypeRe.FindStringIndex(restOfText)
	fmt.Printf("Next type found at:%d\n", nextLoc[1])
	var endPos int
	if nextLoc != nil {
		// Found another Type line.
		endPos = loc[1] + nextLoc[0]
	} else {
		// No other Type found
		endPos = len(inputContent)
	}

	// Extract the text between the Type lines
	textBetweenTypes := inputContent[loc[1]:endPos]

	// Use the existing pattern to match entries
	regex, err := regexp.Compile(tagRecordRegex)
	if err != nil {
		return err
	}

	// Find all matches in the content.
	matches := regex.FindAllStringSubmatch(textBetweenTypes, -1)
	fmt.Printf("%d entries extracted from the tag block.\n", len(matches))

	// Extract patterns to array from the Tag block.
	for _, t := range matches {
		// Check if there is a captured group and extract the content.
		if len(t) > 1 {
			// Ensure there is a captured group.
			tag := strings.TrimSpace(t[2])
			tags = append(tags, tag)
		}
	}

	// Gather categories from the Accounts
	// Compile the regex
	regex, err = regexp.Compile(accountBlockHeaderRegex)
	if err != nil {
		return err
	}
	accountBlocks := regex.FindAllStringSubmatchIndex(inputContent, -1)
	if len(accountBlocks) == 0 {
		fmt.Println("No matches found.")
	}

	// loop over each account block and pull out tags
	for _, accountBlock := range accountBlocks {
		// Find the next Type Block
		accountName := inputContent[accountBlock[2]:accountBlock[3]]

		restOfText := inputContent[accountBlock[1]:]
		nextTypePattern := `(?mi)^\s*!Type:.*$`
		nextTypeRe := regexp.MustCompile(nextTypePattern)
		nextLoc := nextTypeRe.FindStringIndex(restOfText)
		var endPos int
		if nextLoc != nil {
			endPos = accountBlock[1] + nextLoc[0]
		} else {
			endPos = len(inputContent)
		}

		// Extract the text between the type lines
		textBetweenTypes := inputContent[accountBlock[1]:endPos]

		// Use the existing pattern to match entries
		regex, err := regexp.Compile(transactionRegexString)
		if err != nil {
			return nil
		}

		// Find all matches in the content.
		transactions := regex.FindAllStringSubmatch(textBetweenTypes, -1)
		fmt.Printf("%d tags extracted from account: %s\n", len(transactions), accountName)

		// Loop through matches and add categories to the array
		for _, t := range transactions {
			if len(t) > 1 {
				_, tag := splitCategoryAndTag(t[20])
				tag = prepareString(tag)
				tags = append(tags, tag)
			}
		}
	}

	// Sort and dedupe tag list
	outputTagList := sortAndDedupStrings(tags)
	// Write tags to the file
	for _, item := range outputTagList {
		_, err := tagFile.WriteString(item + "\n")
		if err != nil {
			fmt.Printf("Error Writing to tag file:\n")
		}
	}

	fmt.Println("Extracted Tags: ", len(outputTagList))

	return nil
}

func splitCategoryAndTag(originalCategoryValue string) (category string, tag string) {

	var lastItem string = ""
	var rest string = ""

	// Split the string by "/" and remove empty strings
	parts := strings.Split(originalCategoryValue, "/")

	// Filter out empty parts
	var nonEmptyParts []string
	for _, part := range parts {
		if part != "" {
			nonEmptyParts = append(nonEmptyParts, part)
		}
	}

	// If there are no valid parts return empty values
	if len(nonEmptyParts) == 0 {
		return "", ""
	}

	if len(nonEmptyParts) == 1 {
		// Returnm the last element and the rest of the string
		rest = strings.Join(nonEmptyParts[:len(nonEmptyParts)-1], "/")
	} else {
		lastItem = nonEmptyParts[len(nonEmptyParts)-1]
		rest = strings.Join(nonEmptyParts[:len(nonEmptyParts)-1], "/")
	}

	return rest, lastItem
}

func prepareString(s string) string {
	return strings.Replace(s, ",", "", -1)
}

func loadMapping(filePath string) (map[string]string, error) {
	mapping := make(map[string]string)

	file, err := os.OpenFile(filePath, os.O_RDONLY|os.O_CREATE, 0777)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Split the string
		parts := strings.SplitN(line, ",", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid line in mapping file: %s", line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		mapping[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return mapping, nil
}

func applyMapping(input string, mapping map[string]string) string {
	for oldValue, newValue := range mapping {
		input = strings.ReplaceAll(input, oldValue, newValue)
	}
	return input
}

func sortAndDedupStrings(arr []string) []string {
	sort.Strings(arr)

	n := len(arr)
	if n == 0 {
		return arr
	}

	// Deduplication
	deduped := []string{arr[0]}
	for i := 1; i < n; i++ {
		if arr[i] != arr[i-1] {
			deduped = append(deduped, arr[i])
		}
	}

	// Remove Blanks
	var result []string
	for _, str := range deduped {
		if strings.TrimSpace(str) != "" {
			result = append(result, str)
		}
	}
	return result
}
