package bond

import (
	"context"

	"github.com/nokia/srlinux-ndk-go/ndk"
	"google.golang.org/protobuf/encoding/prototext"
)

// ReceiveNetworkInstanceNotifications starts an network instance notification
// stream and sends notifications to channel `NwInst`.
// If the main execution intends to continue running after calling this method,
// it should be called as a goroutine.
// `NwInst` chan carries values of type ndk.NetworkInstanceNotification
func (a *Agent) ReceiveNetworkInstanceNotifications(ctx context.Context) {
	defer close(a.Notifications.NwInst)
	nwInstStream := a.startNwInstNotificationStream(ctx)

	for nwInstStreamResp := range nwInstStream {
		b, err := prototext.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(nwInstStreamResp)
		if err != nil {
			a.logger.Infof("Network instance notification Marshal failed: %+v", err)
			continue
		}

		a.logger.Infof("Received network instance notifications:\n%s", b)

		for _, n := range nwInstStreamResp.GetNotifications() {
			nwInstNotif := n.GetNetworkInstance()
			if nwInstNotif == nil {
				a.logger.Infof("Empty network instance notification:%+v", n)
				continue
			}
			a.Notifications.NwInst <- nwInstNotif
		}
	}
}

// startNwInstNotificationStream starts a notification stream for Network Instance service notifications.
func (a *Agent) startNwInstNotificationStream(ctx context.Context) chan *ndk.NotificationStreamResponse {
	streamID := a.createNotificationStream(ctx)

	a.logger.Info("Network Instance notification stream created", "stream-id", streamID)

	a.addNwInstSubscription(ctx, streamID)

	streamChan := make(chan *ndk.NotificationStreamResponse)
	go a.startNotificationStream(ctx, streamID,
		"nwinst", streamChan)

	return streamChan
}

// addNwInstSubscription adds a subscription for Network Instance service notifications
// to the allocated notification stream.
func (a *Agent) addNwInstSubscription(ctx context.Context, streamID uint64) {
	// create notification register request for nwinst service
	// using acquired stream ID
	notificationRegisterReq := &ndk.NotificationRegisterRequest{
		Op:       ndk.NotificationRegisterRequest_OPERATION_ADD_SUBSCRIPTION,
		StreamId: streamID,
		SubscriptionTypes: &ndk.NotificationRegisterRequest_NetworkInstance{ // nwinst service
			NetworkInstance: &ndk.NetworkInstanceSubscriptionRequest{},
		},
	}

	registerResp, err := a.stubs.sdkMgrService.NotificationRegister(ctx, notificationRegisterReq)
	if err != nil || registerResp.GetStatus() != ndk.SdkMgrStatus_SDK_MGR_STATUS_SUCCESS {
		a.logger.Printf("agent %s failed registering to notification with req=%+v: %v",
			a.Name, notificationRegisterReq, err)
	}
}
