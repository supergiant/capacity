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

// UpdateWorkerReader is a Reader for the UpdateWorker structure.
type UpdateWorkerReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *UpdateWorkerReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {

	case 200:
		result := NewUpdateWorkerOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil

	default:
		return nil, runtime.NewAPIError("unknown error", response, response.Code())
	}
}

// NewUpdateWorkerOK creates a UpdateWorkerOK with default headers values
func NewUpdateWorkerOK() *UpdateWorkerOK {
	return &UpdateWorkerOK{}
}

/*UpdateWorkerOK handles this case with default header values.

workerResponse contains a worker representation.
*/
type UpdateWorkerOK struct {
	Payload *models.Worker
}

func (o *UpdateWorkerOK) Error() string {
	return fmt.Sprintf("[PATCH /api/v1/workers/{machineID}][%d] updateWorkerOK  %+v", 200, o.Payload)
}

func (o *UpdateWorkerOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Worker)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
