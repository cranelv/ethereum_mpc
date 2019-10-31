package step

import "github.com/ethereum/go-ethereum/mpcService/protocol"

type MpcAddressStep struct {
	MpcPoint_Step
}

func CreateMpcAddressStep(result protocol.MpcResultInterface,nodeInfo protocol.MpcNodeInterface) *MpcAddressStep {
	mpc := &MpcAddressStep{MpcPoint_Step: *CreateMpcPoint_Step(result,nodeInfo, []string{protocol.MpcPublicShare}, []string{protocol.PublicKeyResult})}
	return mpc
}

func (addStep *MpcAddressStep) FinishStep( mpc protocol.MpcManager) error {
	err := addStep.MpcPoint_Step.FinishStep(mpc)
	if err != nil {
		return err
	}

	return mpc.CreateKeystore(addStep.mpcResult,addStep.nodeInfo)
}
