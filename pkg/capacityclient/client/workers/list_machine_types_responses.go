// Code generated by go-swagger; DO NOT EDIT.

package workers

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	strfmt "github.com/go-openapi/strfmt"

	models "github.com/supergiant/capacity/pkg/capacityclient/models"
)

// ListMachineTypesReader is a Reader for the ListMachineTypes structure.
type ListMachineTypesReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListMachineTypesReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {

	case 200:
		result := NewListMachineTypesOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil

	default:
		return nil, runtime.NewAPIError("unknown error", response, response.Code())
	}
}

// NewListMachineTypesOK creates a ListMachineTypesOK with default headers values
func NewListMachineTypesOK() *ListMachineTypesOK {
	return &ListMachineTypesOK{}
}

/*ListMachineTypesOK handles this case with default header values.

machineTypesListResponse contains a list of workers.
*/
type ListMachineTypesOK struct {
	Payload []*models.MachineType
}

func (o *ListMachineTypesOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/machinetypes][%d] listMachineTypesOK  %+v", 200, o.Payload)
}

func (o *ListMachineTypesOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
