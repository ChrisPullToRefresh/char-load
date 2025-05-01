package charunit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.viam.com/rdk/components/board"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/generic"
	"go.viam.com/utils"
	"go.viam.com/utils/rpc"
)

var (
	CharUnitLoad     = resource.NewModel("pulltorefresh", "char-unit", "char-unit-load")
	errUnimplemented = errors.New("unimplemented")
)

func init() {
	resource.RegisterService(generic.API, CharUnitLoad,
		resource.Registration[resource.Resource, *Config]{
			Constructor: newCharUnitCharUnitLoad,
		},
	)
}

type Config struct {
	Board string `json:"board"`
	/*
		Put config attributes here. There should be public/exported fields
		with a `json` parameter at the end of each attribute.

		Example config struct:
			type Config struct {
				Pin   string `json:"pin"`
				Board string `json:"board"`
				MinDeg *float64 `json:"min_angle_deg,omitempty"`
			}

		If your model does not need a config, replace *Config in the init
		function with resource.NoNativeConfig
	*/
}

// Validate ensures all parts of the config are valid and important fields exist.
// Returns implicit dependencies based on the config.
// The path is the JSON path in your robot's config (not the `Config` struct) to the
// resource being validated; e.g. "components.0".
func (cfg *Config) Validate(path string) ([]string, error) {
	deps := []string{cfg.Board}

	if cfg.Board == "" {
		return nil, utils.NewConfigValidationFieldRequiredError(path, "board")
	}
	return deps, nil
}

type charUnitCharUnitLoad struct {
	resource.AlwaysRebuild

	name resource.Name

	logger logging.Logger
	cfg    *Config

	cancelCtx  context.Context
	cancelFunc func()

	b board.Board
}

func newCharUnitCharUnitLoad(ctx context.Context, deps resource.Dependencies, rawConf resource.Config, logger logging.Logger) (resource.Resource, error) {

	return NewCharUnitLoad(ctx, deps, rawConf.ResourceName(), rawConf, logger)

}

func NewCharUnitLoad(ctx context.Context, deps resource.Dependencies, name resource.Name, rawConf resource.Config, logger logging.Logger) (resource.Resource, error) {

	cancelCtx, cancelFunc := context.WithCancel(context.Background())

	conf, err := resource.NativeConfig[*Config](rawConf)
	if err != nil {
		cancelFunc()
		return nil, err
	}

	s := &charUnitCharUnitLoad{
		name:       name,
		logger:     logger,
		cfg:        conf,
		cancelCtx:  cancelCtx,
		cancelFunc: cancelFunc,
	}

	if err := s.Reconfigure(ctx, deps, rawConf); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *charUnitCharUnitLoad) Name() resource.Name {
	return s.name
}

func (s *charUnitCharUnitLoad) NewClientFromConn(ctx context.Context, conn rpc.ClientConn, remoteName string, name resource.Name, logger logging.Logger) (resource.Resource, error) {
	panic("not implemented")
}

func (s *charUnitCharUnitLoad) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	// m.logger.Infof("DoCommand called with cmd=%v", cmd)
	for key, value := range cmd {
		switch key {
		// "TurnThenCenter": "SmallLeft"
		case "char_load":
			s.logger.Infof("DoCommand key=%v", key)
			command := value.(string)
			s.logger.Infof("DoCommand command=%v", command)
			switch command {
			case "start":
				go bioCharProcess(s)
			default:
				return nil, fmt.Errorf("unknown DoCommand value for %v = %v", key, value)
			}

			return nil, nil
		default:
			return nil, fmt.Errorf("unknown DoCommand key = %v ", key)
		}
	}
	return nil, fmt.Errorf("unknown DoCommand command map: %v", cmd)
}

func bioCharProcess(s *charUnitCharUnitLoad) {
	if s.b == nil {
		s.logger.Error("s.b (Board component) is nil inside DoCommand")
		return
	}
	// Get the GPIOPin with pin number 11
	pin, err := s.b.GPIOPinByName("11")
	if err != nil {
		s.logger.Error(err)
		return
	}
	s.logger.Infof("DoCommand fetched the pin successfully")

	// Set the pin to high.
	err = pin.Set(context.Background(), true, nil)
	if err != nil {
		s.logger.Error(err)
		return
	}
	s.logger.Infof("DoCommand set the pin to high successfully")

	time.Sleep(60 * time.Second)
	s.logger.Infof("DoCommand finished sleeping")

	// Set the pin to high.
	err = pin.Set(context.Background(), false, nil)
	if err != nil {
		s.logger.Error(err)
		return
	}
	s.logger.Infof("DoCommand set the pin to low successfully")
}

func (s *charUnitCharUnitLoad) Close(context.Context) error {
	// Put close code here
	s.cancelFunc()
	return nil
}

// Reconfigures the model. Most models can be reconfigured in place without needing to rebuild. If you need to instead create a new instance of the motor, throw a NewMustBuildError.
func (s *charUnitCharUnitLoad) Reconfigure(ctx context.Context, deps resource.Dependencies, rawConf resource.Config) error {
	conf, err := resource.NativeConfig[*Config](rawConf)
	if err != nil {
		return err
	}

	b, err := board.FromDependencies(deps, conf.Board)
	if err != nil {
		return fmt.Errorf("no source camera for transform pipeline  (%s): %w", conf.Board, err)
	}
	s.b = b
	s.logger.Info("board is now configured to ", s.b.Name())
	return nil
}
