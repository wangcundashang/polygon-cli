package cdk

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-errors/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	banana_rollup_manager "github.com/0xPolygon/cdk-contracts-tooling/contracts/banana/polygonrollupmanager"
	elderberry_rollup_manager "github.com/0xPolygon/cdk-contracts-tooling/contracts/elderberry/polygonrollupmanager"
	etrog_rollup_manager "github.com/0xPolygon/cdk-contracts-tooling/contracts/etrog/polygonrollupmanager"
)

const (
	ArgRpcURL = "rpc-url"
	ArgForkID = "fork-id"

	ArgRollupManagerAddress = "rollup-manager-address"

	ArgRollupChainID = "rollup-chain-id"
	ArgRollupID      = "rollup-id"
	ArgRollupAddress = "rollup-address"

	defaultRPCURL = "http://localhost:8545"
	defaultForkId = "12"

	// forks
	blueberry   = uint64(4)
	dragonfruit = uint64(5)
	incaberry   = uint64(6)
	etrog       = uint64(7)
	elderberry  = uint64(9)
	feijoa      = uint64(10)
	banana      = uint64(12)
	durian      = uint64(13)

	contractRequestInterval = 200 * time.Millisecond
)

var (
	knownRollupManagerAddresses = map[string]string{
		"bali":    "0xe2ef6215adc132df6913c8dd16487abf118d1764",
		"cardona": "0x32d33D5137a7cFFb54c5Bf8371172bcEc5f310ff",
		"mainnet": "0x5132a183e9f3cb7c848b0aac5ae0c4f0491b7ab2",
	}
	knownForks = map[string]uint64{
		// "4":           blueberry,
		// "blueberry":   blueberry,
		// "5":           dragonfruit,
		// "dragonfruit": dragonfruit,
		// "6":           incaberry,
		// "incaberry":   incaberry,
		"7":          etrog,
		"etrog":      etrog,
		"9":          elderberry,
		"elderberry": elderberry,
		// "10":          feijoa,
		// "feijoa":      feijoa,
		"12":     banana,
		"banana": banana,
		// "13":          durian,
		// "durian":      durian,
	}
)

var CDKCmd = &cobra.Command{
	Use:   "cdk",
	Short: "Utilities for interacting with CDK networks",
	Args:  cobra.NoArgs,
}

type inputArgs struct {
	rpcURL *string

	forkID *string

	rollupManagerAddress *string

	rollupID      *string
	rollupChainID *string
	rollupAddress *string
}

type parsedCDKArgs struct {
	rpcURL    string
	rpcClient *ethclient.Client
	forkID    uint64
}

type parsedRollupManagerArgs struct {
	rollupManager        rollupManagerContractInterface
	rollupManagerAddress common.Address
}

type parsedRollupArgs struct {
	rollupID      uint32
	rollupChainID uint64
	rollupAddress common.Address
}

var cdkInputArgs = inputArgs{}

func checkAddressArg(argFlagName, address string) error {
	prefix := fmt.Sprintf("invalid flag %s: ", argFlagName)

	if !common.IsHexAddress(address) {
		return errors.New(prefix + "invalid address")
	}

	return nil
}

func (inputArgs *inputArgs) parseCDKArgs(ctx context.Context) (*parsedCDKArgs, error) {
	args := &parsedCDKArgs{}

	args.rpcURL = *inputArgs.rpcURL
	args.rpcClient = mustGetRPCClient(ctx, args.rpcURL)

	if inputArgs.forkID != nil && len(*inputArgs.forkID) > 0 {
		_, found := knownForks[*inputArgs.forkID]
		if !found {
			keys := make([]string, len(knownForks))
			for k := range knownForks {
				keys = append(keys, k)
			}
			slices.Sort(keys)
			v := strings.Join(keys, ", ")
			return nil, fmt.Errorf("invalid forkID. supported forkIDs are %s", v)
		}
		args.forkID = knownForks[*inputArgs.forkID]
	}

	return args, nil
}

func (inputArgs *inputArgs) parseRollupManagerArgs(ctx context.Context, cdkArgs parsedCDKArgs) (*parsedRollupManagerArgs, error) {
	args := &parsedRollupManagerArgs{}

	if knownRollupManagerAddress, found := knownRollupManagerAddresses[*cdkInputArgs.rollupManagerAddress]; found {
		args.rollupManagerAddress = common.HexToAddress(knownRollupManagerAddress)
	} else {
		err := checkAddressArg(ArgRollupManagerAddress, *inputArgs.rollupManagerAddress)
		if err != nil {
			return nil, err
		}
		args.rollupManagerAddress = common.HexToAddress(*cdkInputArgs.rollupManagerAddress)
	}

	switch cdkArgs.forkID {
	case etrog:
		rollupManager, err := etrog_rollup_manager.NewPolygonrollupmanager(args.rollupManagerAddress, cdkArgs.rpcClient)
		if err != nil {
			return nil, err
		}
		args.rollupManager = rollupManager
	case elderberry:
		rollupManager, err := elderberry_rollup_manager.NewPolygonrollupmanager(args.rollupManagerAddress, cdkArgs.rpcClient)
		if err != nil {
			return nil, err
		}
		args.rollupManager = rollupManager
	case banana:
		rollupManager, err := banana_rollup_manager.NewPolygonrollupmanager(args.rollupManagerAddress, cdkArgs.rpcClient)
		if err != nil {
			return nil, err
		}
		args.rollupManager = rollupManager
	}

	return args, nil
}

func (inputArgs *inputArgs) parseRollupArgs(ctx context.Context, rollupManagerArgs *parsedRollupManagerArgs) (*parsedRollupArgs, error) {
	args := &parsedRollupArgs{}

	var rollupChainID uint64
	if cdkInputArgs.rollupChainID != nil && len(*cdkInputArgs.rollupChainID) > 0 {
		rollupChainIDN, err := strconv.ParseInt(*cdkInputArgs.rollupChainID, 10, 64)
		if err != nil || rollupChainIDN < 0 {
			return nil, fmt.Errorf("invalid rollupChainID: %s, it must be a valid uint64", *cdkInputArgs.rollupID)
		}
		rollupChainID = uint64(rollupChainIDN)
	}
	args.rollupChainID = rollupChainID

	args.rollupAddress = common.Address{}
	if inputArgs.rollupAddress != nil && len(*inputArgs.rollupAddress) > 0 {
		err := checkAddressArg(ArgRollupAddress, *inputArgs.rollupAddress)
		if err != nil {
			return nil, err
		}
		args.rollupAddress = common.HexToAddress(*inputArgs.rollupAddress)
	}

	args.rollupID = 0
	if cdkInputArgs.rollupID != nil && len(*cdkInputArgs.rollupID) > 0 {
		rollupIDN, err := strconv.Atoi(*cdkInputArgs.rollupID)
		if err != nil || rollupIDN < 0 {
			return nil, fmt.Errorf("invalid rollupID: %s, it must be a valid uint32", *cdkInputArgs.rollupID)
		}
		args.rollupID = uint32(rollupIDN)
	} else {
		if rollupChainID == 0 && args.rollupAddress == (common.Address{}) {
			return nil, fmt.Errorf("%s, %s, or %s must be provided", ArgRollupID, ArgRollupChainID, ArgRollupAddress)
		}
		if rollupChainID != 0 {
			rollupID, err := rollupManagerArgs.rollupManager.ChainIDToRollupID(nil, rollupChainID)
			if err != nil {
				return nil, err
			}
			args.rollupID = rollupID
		} else if args.rollupAddress != (common.Address{}) {
			rollupID, err := rollupManagerArgs.rollupManager.RollupAddressToID(nil, args.rollupAddress)
			if err != nil {
				return nil, err
			}
			args.rollupID = rollupID
		}
	}

	return args, nil
}

func mustGetRPCClient(ctx context.Context, rpcURL string) *ethclient.Client {
	rpcClient, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to Dial RPC")
	}
	return rpcClient
}

func mustLogJSONIndent(v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	log.Info().Msgf("%s", string(b))
}

func init() {
	// cdk
	cdkInputArgs.rpcURL = CDKCmd.PersistentFlags().String(ArgRpcURL, defaultRPCURL, "The RPC URL of the network containing the CDK contracts")
	cdkInputArgs.forkID = CDKCmd.PersistentFlags().String(ArgForkID, defaultForkId, "The ForkID of the cdk networks")

	// rollup manager
	cdkInputArgs.rollupManagerAddress = rollupManagerCmd.PersistentFlags().String(ArgRollupManagerAddress, "", "The address of the rollup contract")

	// rollup
	cdkInputArgs.rollupID = rollupCmd.PersistentFlags().String(ArgRollupID, "", "The rollup ID")
	cdkInputArgs.rollupChainID = rollupCmd.PersistentFlags().String(ArgRollupChainID, "", "The rollup chain ID")
	cdkInputArgs.rollupAddress = rollupCmd.PersistentFlags().String(ArgRollupAddress, "", "The rollup Address")

	CDKCmd.AddCommand(rollupManagerCmd)
	rollupManagerCmd.AddCommand(rollupCmd)

	rollupManagerCmd.AddCommand(rollupManagerListRollupsCmd)
	rollupManagerCmd.AddCommand(rollupManagerListRollupTypesCmd)
	rollupManagerCmd.AddCommand(rollupManagerInspectCmd)
	rollupManagerCmd.AddCommand(rollupManagerDumpCmd)
	rollupManagerCmd.AddCommand(rollupManagerMonitorCmd)

	rollupCmd.AddCommand(rollupInspectCmd)
	rollupCmd.AddCommand(rollupDumpCmd)
	rollupCmd.AddCommand(rollupMonitorCmd)
}
