# Aether - why should i need another cli tool?

I have been a dev on flow for over 5 years, i started during OpenWorldBuilders in 2020 where me and my team made versus. 

Ever since I have spent some of my time working on tools to make it easier for new devs in this ecosystem to get up and running and play fast. 

`flow-cli` is a wonderful tool (i am a commiter), but it is a very low level tool. Lets say you have a little webapp where you want to deploy a contract and have 2 users that can interact in this webapp together. What do you have to do in order to get this to work with flow-cli

1. terminal1: `flow emulator -v` to start the emulator and show logs
2. terminal2: `flow deploy -n emulator` to deploy contracts to emulator
4. terminal3: `flow dev-wallet` we start the dev wallet
5. terminal5: `npm run start` or whatever you use to start your webapp
6. browser: 
 - open up the react-sdk tutorial page
 - hit connect button
 - create a new user, note down the address

 If you now want to sign a transaction as the user you created in dev-wallet you have to add this user manually to flow.json, it will have the same key as your service-account. 

 If you want to run init transactions for your users before you even start up the webapp, that is not possible in a very easy way with only flow-cli tools. 

 These are quiet a few steps and in my opionion it is a lot of things you have to know in order to get this to work. 
 
 So how do you do this with aether? If you want to have aether start your frontend server for you you have to create an aether.yaml config file in your root folder like this one: https://github.com/bjartek/aether-react-example/blob/main/aether.yaml

 In here you basically just tell aether you want it to run your frontend server for you. If you do not want that because you want that in your editor of choice then you can just skip this step. 

 Then you run `aether` in your root folder in a single terminal and the following will happen. 

 aether will start a TUI and show you a dashboard of the init process. This process does the following
  - starts emulator
  - starts dev-wallet
  - starts the flow-evm-gateway
  - creates all accounts in the deployment block for emulator
  - gives them 1000 flow
  - adds them to dev-wallet so you can use them right away
  - deploys all your contracts
  - runs all init transactions for your users
  - starts your frontend server if you configured it to do so

You can now go to your browser and start working. Your users are already there and you can sign with them in web or in flow-cli or even in aether. The runner pane allows you to see all transactions/scripts in your file system and run them. 

If you ran a transaction in your frontend and want to debug it you can press `s` when viewing it to make it a local template that you can then run again on the same state. 

Aether makes developing on flow-emulator and associated tools a breeze. 