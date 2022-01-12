package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func prompt(text string, validate func(string) error) (string,error){
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(text)
		if str, err := reader.ReadString('\n'); err !=nil{
			return "",err
		} else{
			str = strings.TrimSpace(str)
			if err := validate(str); err !=nil{
				fmt.Printf("\u001b[31mError: %s\u001b[0m\n",err.Error())
			} else{
				return str, nil
			}
		}
	}
}