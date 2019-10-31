//431772609360625664 46
constractCode =[{"constant": true,"inputs": [],"name": "getDepositList","outputs": [{"name": "","type": "address[]"}],"payable": false,"stateMutability": "view","type": "function"},{"constant": true,"inputs": [{"name": "addr","type": "address"}],"name": "getDepositInfo","outputs": [{"name": "","type": "uint256"},{"name": "","type": "bytes"},{"name": "","type": "uint256"}],"payable": false,"stateMutability": "view","type": "function"},{"constant": false,"inputs": [{"name": "nodeID","type": "bytes"}],"name": "valiDeposit","outputs": [],"payable": true,"stateMutability": "payable","type": "function"},{"constant": false,"inputs": [{"name": "nodeID","type": "bytes"}],"name": "minerDeposit","outputs": [],"payable": true,"stateMutability": "payable","type": "function"},{"constant": false,"inputs": [],"name": "withdraw","outputs": [],"payable": false,"stateMutability": "nonpayable","type": "function"},{"constant": false,"inputs": [],"name": "refund","outputs": [],"payable": false,"stateMutability": "nonpayable","type": "function"}];
contractDef = eth.contract(constractCode);
ContractAddr = "0x000000000000000000000000000000000000000A";
coinContract = contractDef.at(ContractAddr);
deposit1 =  coinContract.minerDeposit.getData("0xac9f57e4b8cf21d2961d105f0b042c8031c52fd0a564ebfd113c8ea67f89e31f134a7c9a7fc5eb36ccc8099e9c871f26d3f31bbfe90643011ea6795d6a9273f7");
personal.sendTransaction({from:eth.accounts[0], to:ContractAddr,value: web3.toWei(10), data:deposit1, gas: 1000000},"1111111111");

coinContract.getDepositInfo("0x14b640aeb37fbc1c4653a5c5841221c4cc10cc91")
withdrawData = coinContract.withdraw.getData();
refundData = coinContract.refund.getData();


var nodeID = '0xac9f57e4b8cf21d2961d105f0b042c8031c52fd0a564ebfd113c8ea67f89e31f134a7c9a7fc5eb36ccc8099e9c871f26d3f31bbfe90643011ea6795d6a9273f7';
depositData = coinContract.deposit.getData(nodeID);

constractCode = [{"inputs":[{"name":"_greeting","type":"string"}],"payable":false,"stateMutability":"nonpayable","type":"constructor"},{"constant":true,"inputs":[],"name":"greet","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_newgreeting","type":"string"}],"name":"setGreeting","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[],"name":"kill","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"}]
contractDef = eth.contract(constractCode);
ContractAddr = "0xab6d2a165fa6ee9ca17278fd28f537a31689e63b";
coinContract = contractDef.at(ContractAddr);
