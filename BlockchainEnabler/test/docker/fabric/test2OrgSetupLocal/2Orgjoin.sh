# Initiliaization for CompanyA -> generates the necessary files.
go run main.go init CompanyA --local
# Creation for CompanyA : Instantiates the Network 
go run main.go create -u CompanyA

# Initialization for CompanyB
go run main.go init CompanyB --local
# Creation for CompanyB
go run main.go create -u CompanyB

# Invite the CompanyB to join the network of CompanyA
go run main.go invite -u CompanyA -z ~/.enabler/platform/CompanyB/CompanyB_network1/enabler/CompanyBOrg1_Invite.zip
# CompanyB accepts the invite and joins the network created by CompanyA
go run main.go accept -u CompanyB -z ~/.enabler/platform/CompanyA/CompanyA_network1/enabler/CompanyAOrg1_accept_transfer.zip
