import { Component, OnInit, Inject } from '@angular/core';
import { MatDialog, MatDialogRef, MAT_DIALOG_DATA } from '@angular/material';

@Component({
  selector: 'app-node-types-modal',
  templateUrl: './node-types-modal.component.html',
  styleUrls: ['./node-types-modal.component.scss']
})
export class NodeTypesModalComponent implements OnInit {

  public options: any;
  public allowed: any;
  public provider: string;

  constructor(
    public dialogRef: MatDialogRef<NodeTypesModalComponent>,
      @Inject(MAT_DIALOG_DATA) public data: any
  ) {
    this.provider = this.data.provider;
    this.options = new Set(this.data.options);
    this.allowed = new Set(this.data.allowed);
  }

  toggleOption(event) {
    if (event.checked) {
      this.allowed.add(event.source.name);
    } else {
      this.allowed.delete(event.source.name);
    }
  }

  allOptionsSelected(allowed, options) {
    return allowed.size === options.size && Array.from(options).every(a => allowed.has(a))
  }

  toggleSelectAll(event) {
    if (event.checked) {
      this.allowed = new Set(this.options);
    } else {
      this.allowed.clear();
    }
  }

  onNoClick(): void {
    this.dialogRef.close();
  }

  ngOnInit() {
  }

}
