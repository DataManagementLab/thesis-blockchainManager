# Invite the CompanyB to join the network of CompanyA
go run main.go add -u CompanyA -z ~/.enabler/platform/CompanyB/CompanyB_network1/enabler/CompanyBOrg1_Invite.zip
# CompanyB accepts the invite and joins the network created by CompanyA
go run main.go join -u CompanyB -z ~/.enabler/platform/CompanyA/CompanyA_network1/enabler/CompanyAOrg1_accept_transfer.zip


# Now CompanyB has joined the network created by CompanyA. 
# CompanyA now invites the CompanyC to join its network. However since CompanyB is also part of the CompanyA network, it also need to sign the request.

# Invite the CompanyB to join the network of CompanyA
go run main.go add -u CompanyA -z ~/.enabler/platform/CompanyC/CompanyC_network1/enabler/CompanyCOrg1_Invite.zip
# Now the CompanyB needs to sign and update the CompanyC invite so it can be added to the network.

go run main.go sign -u CompanyB --update -z ~/.enabler/platform/CompanyA/CompanyA_network1/enabler/CompanyCOrg1_sign_transfer.zip


# CompanyC  accepts the invite and joins the network created by CompanyA
go run main.go join -u CompanyC -z ~/.enabler/platform/CompanyA/CompanyA_network1/enabler/CompanyAOrg1_accept_transfer.zip
