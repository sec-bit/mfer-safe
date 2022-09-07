import { ethers } from "ethers";
import sushiTokenList from "./sushi_token_list.json";

export function loadTokenList() {
    var tokenDict = {};
    sushiTokenList.tokens.forEach(token => {
        tokenDict[token.address.toLowerCase()] = {
            "symbol": token.symbol,
            "name": token.name,
            "decimals": token.decimals
        };
    })
    return tokenDict;
}

export function convertToDecimalFormat(amount, decimals) {
    return ethers.utils.formatUnits(amount, decimals);
}

export function processEvents(events) {
    var obj;
    var amount;
    var tokenID;
    var approvalERC20Events = [];
    var approvalERC721Events = [];
    var approvalForAllEvents = [];
    var transferERC20Events = [];
    var transferERC721Events = [];
    var userTokenBalance = {};
    // var callValue = [];
    var processedEvents = {
        "approvalERC20Events": approvalERC20Events,
        "approvalERC721Events": approvalERC721Events,
        "approvalForAllEvents": approvalForAllEvents,
        "transferERC20Events": transferERC20Events,
        "transferERC721Events": transferERC721Events,
        "userTokenBalance": userTokenBalance,
    };

    const namedtopics = {
        "Approval": "0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925",
        "Transfer": "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
        "ApprovalForAll": "0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31",
    }

    for (var i = 0; i < events.length; i++) {
        var event = events[i];
        if (event.topics === undefined || event.topics.length < 1) {
            continue
        }

        switch (event.topics[0]) {
            case namedtopics.Transfer:
                obj = {
                    "token": event.address,
                    "from": "0x" + event.topics[1].slice(-40),
                    "to": "0x" + event.topics[2].slice(-40),
                }
                switch (event.topics.length) {
                    // ERC20
                    case 3:
                        amount = ethers.BigNumber.from(event.data);
                        obj["amount"] = ethers.BigNumber.from(event.data).toString();
                        transferERC20Events.push(obj);
                        userTokenBalance[obj.from] = userTokenBalance[obj.from] || {};
                        userTokenBalance[obj.to] = userTokenBalance[obj.to] || {};
                        userTokenBalance[obj.from][obj.token] = userTokenBalance[obj.from][obj.token] || ethers.BigNumber.from(0);
                        userTokenBalance[obj.to][obj.token] = userTokenBalance[obj.to][obj.token] || ethers.BigNumber.from(0);
                        userTokenBalance[obj.from][obj.token] = userTokenBalance[obj.from][obj.token].sub(amount);
                        userTokenBalance[obj.to][obj.token] = userTokenBalance[obj.to][obj.token].add(amount);
                        break;
                    // ERC721
                    case 4:
                        tokenID = ethers.BigNumber.from(event.topics[3])
                        // assume token id is hash, use hex format
                        obj["tokenID"] = tokenID.shr(128).eq(0) ? tokenID.toString() : event.topics[3]
                        transferERC721Events.push(obj);
                        break;
                    default:
                }
                break;
            case namedtopics.Approval:
                obj = {
                    "token": event.address,
                    "owner": "0x" + event.topics[1].slice(-40),
                    "spender": "0x" + event.topics[2].slice(-40),
                }
                switch (event.topics.length) {
                    // ERC20
                    case 3:
                        amount = ethers.BigNumber.from(event.data);
                        // assume above 2**254 is infinite
                        obj["amount"] = amount.shr(254).eq(0) ? ethers.BigNumber.from(event.data).toString() : "infinite";
                        approvalERC20Events.push(obj);
                        break;
                    // ERC721
                    case 4:
                        tokenID = ethers.BigNumber.from(event.topics[3])
                        // assume token id is hash, use hex format
                        obj["tokenID"] = tokenID.shr(128).eq(0) ? tokenID.toString() : event.topics[3]
                        approvalERC721Events.push(obj);
                        break;
                    default:
                }
                break;
            case namedtopics.ApprovalForAll:
                obj = {
                    "token": event.address,
                    "owner": "0x" + event.topics[1].slice(-40),
                    "operator": "0x" + event.topics[2].slice(-40),
                    "approved": event.data !== "0x0000000000000000000000000000000000000000000000000000000000000000"? "YES" : "NO",
                }
                approvalForAllEvents.push(obj);
                break;
            default:
        }
    }
    return processedEvents;
}