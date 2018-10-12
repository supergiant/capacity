import { Component, OnInit, Inject } from '@angular/core';
import { MatDialog, MatDialogRef, MAT_DIALOG_DATA } from '@angular/material';

@Component({
  selector: 'app-confirm-delete-modal',
  templateUrl: './confirm-delete-modal.component.html',
  styleUrls: ['./confirm-delete-modal.component.scss']
})
export class ConfirmDeleteModalComponent implements OnInit {

  public name: string;

  constructor(
    public dialogRef: MatDialogRef<ConfirmDeleteModalComponent>,
      @Inject(MAT_DIALOG_DATA) public data: any
  ) { this.name = this.data.name }

  ngOnInit() {
  }

}
