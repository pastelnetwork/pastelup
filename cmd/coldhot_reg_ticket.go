package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pastelnetwork/gonode/common/log"
)

const (
	minBalanceForTicketReg = 1000.0
)

func (r *ColdHotRunner) getBalance(ctx context.Context, cold bool) (balance float64, err error) {
	var out string
	var output []byte
	if cold {
		out, err = RunPastelCLI(ctx, r.config, "getbalance")
	} else {
		cmd := fmt.Sprintf("%s %s", r.opts.remotePastelCli, "getbalance")
		output, err = r.sshClient.Cmd(cmd).Output()
		out = string(output)
	}

	if err != nil {
		log.WithContext(ctx).WithError(err).WithField("out", string(out)).
			Error("Failed to get balance")

		return 0.0, err
	}
	strBalance := strings.TrimSpace(strings.Trim(out, "\n"))

	return strconv.ParseFloat(strBalance, 64)
}

func (r *ColdHotRunner) getRemotePastelAddr(ctx context.Context) (addr string, err error) {
	cmd := fmt.Sprintf("%s %s", r.opts.remotePastelCli, `getaccountaddress ""`)
	out, err := r.sshClient.Cmd(cmd).SmartOutput()
	if err != nil {
		log.WithContext(ctx).WithError(err).WithField("out", string(out)).
			Error("Failed to get remote addr")
		return "", err
	}

	return strings.TrimSpace(strings.Trim(string(out), "\n")), nil
}

func (r *ColdHotRunner) sendAmountToAddrFromLocal(ctx context.Context, zcashAddr string, amount float64) (txid string, err error) {
	out, err := RunPastelCLI(ctx, r.config, "sendtoaddress", zcashAddr, fmt.Sprintf("%v", amount))
	if err != nil {
		log.WithContext(ctx).WithError(err).WithField("out", string(out)).
			WithField("addr", zcashAddr).WithField("amount", amount).
			Error("Failed to send amount to address")
		return "", err
	}

	return strings.TrimSpace(strings.Trim(string(out), "\n")), nil
}

func (r *ColdHotRunner) handleAskForBalance(ctx context.Context, remoteBalance float64, localBalance float64, address string) (bool, error) {
	if ok, _ := AskUserToContinue(ctx, fmt.Sprintf(` Neither remote (%v PSL) nor
	local (%v PSL)  has enough balance.\n Would you like to trasnfer balance from any of your wallet to an address & continue? Y/N`,
		remoteBalance, localBalance)); !ok {
		return false, nil
	}

	if ok, _ := AskUserToContinue(ctx, fmt.Sprintf(`Please send 1,000 PSL coins to this address: %s  -- 
	and press Y to continue. Or N to cancel. Y/N`, address)); !ok {
		return false, nil
	}

	return r.handleWaitForBalance(ctx)
}

func (r *ColdHotRunner) handleWaitForBalance(ctx context.Context) (bool, error) {
	i := 0
	for {
		fmt.Println("checking remote for balance...")
		remoteBalance, err := r.getBalance(ctx, false)
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("unable to get remote balance")
			return false, fmt.Errorf("handleWaitForBalance: getRemoteBalance: %s", err)
		}

		if remoteBalance >= minBalanceForTicketReg {
			return true, nil
		}
		time.Sleep(6 * time.Second)

		if i == 9 {
			if ok, _ := AskUserToContinue(ctx, `Enough balance not recieved yet. 
			Would you like to continue & wait? Y/N`); !ok {
				return false, nil
			}
			i = 0
		}
		i++
	}
}

func (r *ColdHotRunner) handleTransferBalance(ctx context.Context, remoteBalance float64, localBalance float64, addr string) (txid string, err error) {
	yes, _ := AskUserToContinue(ctx, fmt.Sprintf(`Remote Node does not have enough balance (%v PSL) but 
	your local one does! (%v PSL) \\nDo you want us to transfer 1,000 PSL from your local to remote
	 & proceed with ticket register? Y/N`,
		remoteBalance, localBalance))

	if !yes {
		return txid, nil
	}

	txid, err = r.sendAmountToAddrFromLocal(ctx, addr, minBalanceForTicketReg)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("unable to send amount from local to remote")
		return txid, fmt.Errorf("sendAmountToAddrFromLocal: %s", err)
	}

	return txid, nil
}

func (r *ColdHotRunner) registerTicketPastelID(ctx context.Context) (err error) {
	cmd := fmt.Sprintf("%s %s %s %s", r.opts.remotePastelCli, "tickets register mnid",
		flagMasterNodePastelID, flagMasterNodePassPhrase)

	remoteBalance, err := r.getBalance(ctx, false)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("unable to get remote balance")
		return fmt.Errorf("getRemoteBalance: %s", err)
	}

	log.WithContext(ctx).WithField("balance", remoteBalance).Info("got remote balance")

	localBalance, err := r.getBalance(ctx, true)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("unable to get local balance")
		return fmt.Errorf("getlocalBalance: %s", err)
	}
	log.WithContext(ctx).WithField("balance", localBalance).Info("got local balance")

	addr, err := r.getRemotePastelAddr(ctx)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("unable to get remote pastel address")
		return fmt.Errorf("get remote pastel addr: %s", err)
	}

	var balanceEnough bool
	var txid string
	if remoteBalance < minBalanceForTicketReg && localBalance < minBalanceForTicketReg {
		balanceEnough, err = r.handleAskForBalance(ctx, remoteBalance, localBalance, addr)
		if err != nil {
			return fmt.Errorf("handleWaitForBalance: %s", err)
		}
	} else if remoteBalance < minBalanceForTicketReg && localBalance > minBalanceForTicketReg {
		txid, err = r.handleTransferBalance(ctx, remoteBalance, localBalance, addr)
		if err != nil {
			return fmt.Errorf("handleTransferBalance: %s", err)
		}

		balanceEnough, err = r.handleWaitForBalance(ctx)
		if err != nil {
			return fmt.Errorf("handleWaitForBalance after transfer: %s", err)
		}
	} else {
		balanceEnough = true
	}
	if !balanceEnough && txid == "" {
		fmt.Println("please execute the following command when remote has enough balance.")
		fmt.Println("\n*******************************************************************")
		fmt.Println(cmd)
		fmt.Println("*******************************************************************")
		return nil
	}

	out, err := r.sshClient.Cmd(cmd).Output()
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to register ticket mnid")
		fmt.Println("please execute the following command when remote has enough balance.")
		fmt.Println("\n*******************************************************************")
		fmt.Println(cmd)
		fmt.Println("*******************************************************************")
		return err
	}

	/*var pastelidSt structure.RPCPastelID
	if err = json.Unmarshal([]byte(pastelid), &pastelidSt); err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to parse pastelid json")
		return err
	}
	flagMasterNodePastelID = pastelidSt.Pastelid*/

	log.WithContext(ctx).Infof("Register ticket pastelid result = %s", string(out))
	return nil
}
