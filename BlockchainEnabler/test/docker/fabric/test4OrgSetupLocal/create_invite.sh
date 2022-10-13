go run main.go create -u ashwin
go run main.go create -u muhammad

go run main.go invite -u kinshuk -z ~/.enabler/platform/ashwin/ashwin_network1/enabler/ashwinOrg1_Invite.zip

go run main.go accept -u ashwin -z ~/.enabler/platform/kinshuk/kinshuk_network1/enabler/kinshukOrg1_accept_transfer.zip

go run main.go invite -u kinshuk -z ~/.enabler/platform/muhammad/muhammad_network1/enabler/muhammadOrg1_Invite.zip 

go run main.go sign -u ashwin --update -z ~/.enabler/platform/kinshuk/kinshuk_network1/enabler/muhammadOrg1_sign_transfer.zip

go run main.go accept -u muhammad -z ~/.enabler/platform/kinshuk/kinshuk_network1/enabler/kinshukOrg1_accept_transfer.zip

go run main.go init simon -s

go run main.go invite -u kinshuk -z ~/.enabler/platform/simon/simon_network1/enabler/simonOrg1_Invite.zip

go run main.go sign -u ashwin -z ~/.enabler/platform/kinshuk/kinshuk_network1/enabler/simonOrg1_sign_transfer.zip

go run main.go accept -u simon -z ~/.enabler/platform/kinshuk/kinshuk_network1/enabler/kinshukOrg1_accept_transfer.zip

go run main.go sign -u muhammad --update -z ~/.enabler/platform/ashwin/ashwin_network1/enabler/simonOrg1_sign_transfer.zip

