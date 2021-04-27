package handlers

import (
	"context"
	"fmt"

	"github.com/Pylons-tech/pylons/x/pylons/keep"
	"github.com/Pylons-tech/pylons/x/pylons/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// CreateTrade is used to create a trade by a user
func (k msgServer) CreateTrade(ctx context.Context, msg *types.MsgCreateTrade) (*types.MsgCreateTradeResponse, error) {

	err := msg.ValidateBasic()
	if err != nil {
		return nil, errInternal(err)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)

	for _, tii := range msg.ItemInputs {
		_, err := k.GetCookbook(sdkCtx, tii.CookbookID)
		if err != nil {
			return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, fmt.Sprintf("You specified a cookbook that does not exist where raw error is %+v", err))
		}
	}

	for _, item := range msg.ItemOutputs {
		itemFromStore, err := k.GetItem(sdkCtx, item.ID)
		if err != nil {
			return nil, errInternal(err)
		}
		if itemFromStore.Sender != msg.Sender {
			return nil, sdkerrors.Wrap(sdkerrors.ErrUnauthorized, fmt.Sprintf("item with %s id is not owned by sender", item.ID))
		}
		if err = itemFromStore.NewTradeError(); err != nil {
			return nil, errInternal(fmt.Errorf("%s item id is not tradable", itemFromStore.ID))
		}
	}

	if !keep.HasCoins(k.Keeper, sdkCtx, sender, msg.CoinOutputs) {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInsufficientFunds, "sender doesn't have enough coins for the trade")
	}

	err = k.LockCoin(sdkCtx, types.NewLockedCoin(sender, msg.CoinOutputs))

	if err != nil {
		return nil, errInternal(err)
	}

	trade := types.NewTrade(msg.ExtraInfo,
		msg.CoinInputs,
		msg.ItemInputs,
		msg.CoinOutputs,
		msg.ItemOutputs,
		sender)
	if err := k.SetTrade(sdkCtx, trade); err != nil {
		return nil, errInternal(err)
	}

	// set items' owner trade id
	for _, item := range msg.ItemOutputs {
		itemFromStore, err := k.GetItem(sdkCtx, item.ID)
		if err != nil {
			return nil, errInternal(err)
		}
		itemFromStore.OwnerTradeID = trade.ID
		err = k.SetItem(sdkCtx, itemFromStore)
		if err != nil {
			return nil, errInternal(err)
		}
	}

	return &types.MsgCreateTradeResponse{
		TradeID: trade.ID,
		Message: "successfully created a trade",
		Status:  "Success",
	}, nil
}
