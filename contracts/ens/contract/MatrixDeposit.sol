pragma solidity ^0.4.0;

contract Owned {
    address public owner;
    modifier onlyOwner {
        require(msg.sender == owner);
        _;
    }

    function Owned() public {
        owner = msg.sender;
    }

    function changeOwner(address newOwner) onlyOwner public {
        owner = newOwner;
    }
}
library SafeMath {

    function mul(uint256 a, uint256 b) internal constant returns (uint256) {
        uint256 c = a * b;
        assert(a == 0 || c / a == b);
        return c;
    }

    function div(uint256 a, uint256 b) internal constant returns (uint256) {
        // assert(b > 0); // Solidity automatically throws when dividing by 0
        uint256 c = a / b;
        // assert(a == b * c + a % b); // There is no case in which this doesn't hold
        return c;
    }

    function sub(uint256 a, uint256 b) internal constant returns (uint256) {
        assert(b <= a);
        return a - b;
    }

    function add(uint256 a, uint256 b) internal constant returns (uint256) {
        uint256 c = a + b;
        assert(c >= a);
        return c;
    }

}
contract MatrixDeposit is Owned {
    using SafeMath for uint256;
    enum StatePhases { deposit,elect, withdraw ,refund}
    struct depInfo {
        uint deposit;
        bytes NodeID;
        uint withDrawHeight;
        StatePhases state;
    }
    uint totalDeposit;
    /** the number of stake holders **/
    uint public numDepositors;
    /** List of all Depositors **/
    address[] public Depositors;
    /** Deposit per user address **/
    mapping(address => depInfo) public mapDeposites;

    function deposit(bytes nodeID) payable public{
        require(msg.value > 0);     // Check that something has been sent
        totalDeposit += msg.value;
        addDepositor(msg.sender,nodeID,msg.value);
    }
    /**
* Adds a new depositor to the list.
* @param _from the address of the depositor
*        nodeID  the current depositor' nodeID
**/
    function addDepositor(address _from,bytes nodeID, uint value) internal{
        depInfo storage dep = mapDeposites[_from];
        require(dep.state == StatePhases.deposit || dep.state == StatePhases.elect);
        if(dep.deposit == 0){
            if(numDepositors < Depositors.length){
                Depositors[numDepositors] = _from;
            }
            else{
                Depositors.push(_from);
            }
            numDepositors++;
        }
        dep.deposit += value;
        dep.NodeID = nodeID;
        dep.state = StatePhases.deposit;
        mapDeposites[_from] = dep;
        if(numDepositors < Depositors.length)
            Depositors[numDepositors] = _from;
        else
            Depositors.push(_from);
    }

    function elect() public {

    }
    function withDraw() public{
        depInfo storage dep = mapDeposites[msg.sender];
        require(dep.deposit > 0 );
        require(dep.state == StatePhases.deposit);
        dep.state = StatePhases.withdraw;
        dep.withDrawHeight = block.number;
    }
    function refund() public{
        depInfo storage dep = mapDeposites[msg.sender];
        require(dep.deposit > 0 );
        require(dep.state == StatePhases.withdraw);
        require(dep.withDrawHeight+300 < block.number);
        msg.sender.transfer(dep.deposit);
        totalDeposit -= dep.deposit;
        dep.deposit = 0;
        dep.state = StatePhases.refund;
        dep.withDrawHeight = 0;
        mapDeposites[msg.sender] = dep;
        for (uint index = 0; index<numDepositors; index++) {
            if (Depositors[index] == msg.sender){
                removeHolder(msg.sender,index);
                break;
            }
        }

    }
    /**
    * Removes a depositor from the list.
    * @param _from the address of the depositor
    *        index  the index of the depositor
    **/
    function removeHolder(address _from, uint index) internal{
        require(Depositors[index] == _from);
        numDepositors -= 1;
        Depositors[index] = Depositors[numDepositors];
    }

}