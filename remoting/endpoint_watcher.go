package remoting

import (
	"log"

	"github.com/AsynkronIT/protoactor-go/actor"
)

func newEndpointWatcher(address string) actor.Producer {
	return func() actor.Actor {
		return &endpointWatcher{
			address: address,
		}
	}
}

type endpointWatcher struct {
	address string
	watched map[string]*actor.PID //key is the watching PID string, value is the watched PID
	watcher map[string]*actor.PID //key is the watched PID string, value is the watching PID
}

func (state *endpointWatcher) initialize() {
	log.Printf("[REMOTING] Started EndpointWatcher for address %v", state.address)
	state.watched = make(map[string]*actor.PID)
	state.watcher = make(map[string]*actor.PID)
}

func (state *endpointWatcher) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		state.initialize()

	case *remoteTerminate:
		delete(state.watched, msg.Watcher.Id)
		delete(state.watcher, msg.Watchee.Id)

	case *EndpointTerminated:

		log.Printf("[REMOTING] EndpointWatcher handling terminated address %v", msg.Address)

		for id, pid := range state.watched {

			//try to find the watcher ID in the local actor registry
			ref, ok := actor.ProcessRegistry.GetLocal(id)
			if ok {

				//create a terminated event for the Watched actor
				terminated := &actor.Terminated{
					Who:               pid,
					AddressTerminated: true,
				}

				watcher := actor.NewLocalPID(id)
				//send the address Terminated event to the Watcher
				ref.SendSystemMessage(watcher, terminated)
			}
		}

		ctx.Become(state.Terminated)

	case *remoteWatch:

		state.watched[msg.Watcher.Id] = msg.Watchee
		state.watcher[msg.Watchee.Id] = msg.Watcher

		//recreate the Watch command
		w := &actor.Watch{
			Watcher: msg.Watcher,
		}

		//pass it off to the remote PID
		sendRemoteMessage(msg.Watchee, w, nil)

	case *remoteUnwatch:

		//delete the watch entries
		delete(state.watched, msg.Watcher.Id)
		delete(state.watcher, msg.Watchee.Id)

		//recreate the Unwatch command
		uw := &actor.Unwatch{
			Watcher: msg.Watcher,
		}

		//pass it off to the remote PID
		sendRemoteMessage(msg.Watchee, uw, nil)

	default:
		log.Printf("[REMOTING] EndpointWatcher for %v, Unknown message %v", state.address, msg)
	}
}

func (state *endpointWatcher) Terminated(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *remoteTerminate:
	//pass
	case *EndpointTerminated:
	//pass
	case *remoteWatch:

		//try to find the watcher ID in the local actor registry
		ref, ok := actor.ProcessRegistry.GetLocal(msg.Watcher.Id)
		if ok {

			//create a terminated event for the Watched actor
			terminated := &actor.Terminated{
				Who:               msg.Watchee,
				AddressTerminated: true,
			}

			//send the address Terminated event to the Watcher
			ref.SendSystemMessage(msg.Watcher, terminated)
		}

	case *remoteUnwatch:
	//pass

	default:
		log.Printf("[REMOTING] EndpointWatcher for %v, Unknown message %v", state.address, msg)
	}
}
